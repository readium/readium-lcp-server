// Copyright 2025 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/pack"
	uuid "github.com/satori/go.uuid"
)

// Publication aggregates information during the process
type Publication struct {
	UUID           string
	Title          string
	Date           string
	Description    string
	Language       []string
	Publisher      []string
	Author         []string
	Subject        []string
	CoverExtension string
	CoverUrl       string
	StorageMode    int
	FileName       string
	EncryptionKey  []byte
	Location       string
	ContentType    string
	Size           uint32
	Checksum       string
}

// ProcessEncryption encrypts a publication
// inputPath must contain a processable file extension.
func ProcessEncryption(contentID, contentKey, inputPath, tempRepo, outputRepo, storageRepo, storageURL, storageFilename string, extractCover bool) (*Publication, error) {

	if inputPath == "" {
		return nil, errors.New("ProcessEncryption, parameter error")
	}
	log.Println("Process ", inputPath)

	var pub Publication

	// if contentID is not set, generate a random UUID
	if contentID == "" {
		uid, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		contentID = uid.String()
	}
	pub.UUID = contentID

	// create a temp folder if declared, or use the current dir
	if tempRepo != "" {
		log.Println("Create the temp folder ", tempRepo)
		err := os.MkdirAll(tempRepo, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return nil, err
		}
	} else {
		tempRepo, _ = os.Getwd()
	}

	// if the input file is stored on a remote server, fetch it and store it into a temp folder
	tempPath, err := fetchInputFile(inputPath, tempRepo, contentID)
	if err != nil {
		return nil, err
	}
	deleteTemp := false
	// if a temp file has been fetched, it will be deleted later
	if tempPath != "" {
		deleteTemp = true
		inputPath = tempPath
	}

	// select a storage mode
	pub.StorageMode = apilcp.Storage_none
	// if the storage repo is set, set storage mode and output repository
	// note: the -storage parameter takes precedence over -output
	if storageRepo != "" {
		// S3 storage is specified by the presence of "s3:" at the start of the -storage param
		if strings.HasPrefix(storageRepo, "s3:") {
			pub.StorageMode = apilcp.Storage_s3
			outputRepo = tempRepo // before move to s3
		} else {
			// file system storage
			pub.StorageMode = apilcp.Storage_fs
			// create the storage folder when necessary
			err := os.MkdirAll(storageRepo, os.ModePerm)
			if err != nil && !os.IsExist(err) {
				return nil, err
			}
			// the encrypted file will be directly generated inside the storage path
			outputRepo = storageRepo
		}
	}
	// if the output repo is still not set, use the temp directory.
	if outputRepo == "" {
		outputRepo = tempRepo
	}

	// set target file info
	targetFileInfo(&pub, inputPath, storageFilename)

	// set the target file name; use the content id by default
	if storageFilename == "" {
		storageFilename = pub.UUID
	}

	// set the output path
	outputPath := filepath.Join(outputRepo, storageFilename)

	// define an AES encrypter
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// select the encryption process from the input file extension
	err = nil

	inputExt := filepath.Ext(inputPath)

	// the cover can only be extracted if lcpencrypt stores the file (not if the LCP Server store the file)
	//if storageRepo == "" {
	//	extractCover = false
	//}

	switch inputExt {
	case ".epub":
		err = processEPUB(&pub, inputPath, outputPath, encrypter, contentKey, extractCover)
	case ".pdf":
		err = processPDF(&pub, inputPath, outputPath, encrypter, contentKey) // a cover is always extracted from the PDF
	case ".lpf":
		err = processLPF(&pub, inputPath, outputPath, encrypter, contentKey, extractCover)
	case ".audiobook", ".divina", ".webpub", ".rpf":
		err = processRPF(&pub, inputPath, outputPath, encrypter, contentKey, extractCover)
	default:
		return nil, errors.New("unprocessable extension " + inputExt)
	}
	if err != nil {
		return nil, err
	}

	if deleteTemp {
		log.Println("Delete the temp file ", inputPath)
		err = os.Remove(inputPath)
		if err != nil {
			return nil, err
		}
	}

	// store the publication if required, and set pub.Location
	var mode string
	switch pub.StorageMode {
	case apilcp.Storage_none:
		// the license server will have to store the encrypted publication
		// warning: the license server must have read access to the output repo.
		// location indicates to the license server the path to the encrypted publication
		pub.Location = outputPath
		mode = "temp"
	case apilcp.Storage_fs:
		// the encryption tools stores the encrypted publication in a file system
		// location indicates the url of the publication
		pub.Location, err = url.JoinPath(storageURL, storageFilename)
		if err != nil {
			return nil, err
		}
		mode = "file system"
	case apilcp.Storage_s3:
		// the encryption tool stores the encrypted publication in an S3 storage
		// store the encrypted file in its definitive S3 storage, delete the input file
		err = StoreFileOnS3(outputPath, storageRepo, storageFilename)
		if err != nil {
			return nil, err
		}
		// if a cover was extracted (pub.CoverExtension not empty), store it in S3 too.
		// add the cover extension to the storageFilename
		// delete the cover file after storage
		if extractCover && pub.CoverExtension != "" {
			err = StoreFileOnS3(outputPath+pub.CoverExtension, storageRepo, storageFilename+pub.CoverExtension)
			if err != nil {
				return nil, err
			}
		}

		// location indicates the url of the publication on S3
		pub.Location, err = url.JoinPath(storageURL, storageFilename)
		if err != nil {
			return nil, err
		}
		mode = "s3"
	}
	log.Println("Storage mode", mode, "- location", pub.Location)

	// if a cover was extracted, set its url
	if extractCover && pub.CoverExtension != "" {
		pub.CoverUrl, _ = url.JoinPath(storageURL, storageFilename+pub.CoverExtension)
		log.Println("Cover Url", pub.CoverUrl)
	}

	return &pub, nil
}

// fetchInputFile fetches the input file from a remote server
func fetchInputFile(inputPath, tempRepo, contentID string) (string, error) {

	if inputPath == "" || tempRepo == "" || contentID == "" {
		return "", errors.New("fetchInputFile, parameter error")
	}

	url, err := url.Parse(inputPath)
	if err != nil {
		// this is not a valid URL
		return "", nil
	}

	// no need to fetch the file, which is in a file system
	if url.Scheme != "http" && url.Scheme != "https" && url.Scheme != "ftp" {
		return "", nil
	}

	log.Println("Create a temporary file and fetch the input file")

	// the temp file has the same extension as the remote file
	inputExt := filepath.Ext(inputPath)
	tempPath := filepath.Join(tempRepo, contentID+inputExt)
	// create the temp file
	out, err := os.Create(tempPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// fetch the file
	switch url.Scheme {
	case "http", "https":
		res, err := http.Get(inputPath)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		defer out.Close()
		_, err = io.Copy(out, res.Body)
		if err != nil {
			return "", err
		}
	case "ftp":
		// we'll use https://github.com/jlaffaye/ftp when requested
		return "", errors.New("ftp not supported yet")
	}
	return tempPath, nil
}

// targetFileInfo sets the file name and content type
// which will be used during future downloads
func targetFileInfo(pub *Publication, inputPath, storageFilename string) error {

	// if the storage filename was imposed, use it
	if storageFilename != "" {
		pub.FileName = storageFilename
	} else {
		//  generate a filename from the input filename and a target extension
		inputFile := filepath.Base(inputPath)
		inputExt := filepath.Ext(inputPath)
		fileNameNoExt := inputFile[:len(inputFile)-len(inputExt)]

		var ext string
		switch inputExt {
		case ".epub":
			ext = ".epub"
		case ".pdf":
			ext = ".lcpdf"
		case ".audiobook", ".rpf":
			ext = ".lcpau"
		case ".divina":
			ext = ".lcpdi"
		case ".lpf":
			// short term solution. We'll need to inspect the W3C manifest and check conformsTo,
			// to be certain this is an audiobook (vs another profile of Web Publication)
			ext = ".lcpau"
		case ".webpub":
			// short term solution. We'll need to inspect the RWP manifest and check conformsTo,
			// to be certain this package contains a pdf
			ext = ".lcpdf"
		}
		pub.FileName = fileNameNoExt + ext
	}

	// find the target mime type
	outputExt := filepath.Ext(pub.FileName)
	switch outputExt {
	case ".epub":
		pub.ContentType = epub.ContentType_EPUB
	case ".lcpdf":
		pub.ContentType = "application/pdf+lcp"
	case ".lcpau":
		pub.ContentType = "application/audiobook+lcp"
	case ".lcpdi":
		pub.ContentType = "application/divina+lcp"
	}
	return nil
}

// checksum calculates the checksum of a file
func checksum(file *os.File) string {

	hasher := sha256.New()
	file.Seek(0, 0)
	if _, err := io.Copy(hasher, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// processEPUB generates an encrypted EPUB
func processEPUB(pub *Publication, inputPath string, outputPath string, encrypter crypto.Encrypter, contentKey string, extractCover bool) error {

	log.Println("Process as EPUB")

	// create a zip reader from the input path
	zr, err := zip.OpenReader(inputPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	// generate an EPUB object
	epub, err := epub.Read(&zr.Reader)
	if err != nil {
		return err
	}

	// set publication metadata
	pub.Title = epub.Package[0].Metadata.Title[0]
	pub.Date = epub.Package[0].Metadata.Date
	pub.Description = epub.Package[0].Metadata.Description
	pub.Language = epub.Package[0].Metadata.Language
	pub.Publisher = epub.Package[0].Metadata.Publisher
	pub.Author = epub.Package[0].Metadata.Author
	pub.Subject = epub.Package[0].Metadata.Subject

	// look for the cover image, in case its extraction is requested
	coverImageID := "cover-image"
	for _, meta := range epub.Package[0].Metadata.Metas {
		if meta.Name == "cover" {
			coverImageID = meta.Content
		}
	}
	var coverPath string
	for _, item := range epub.Package[0].Manifest.Items {
		if strings.Contains(item.Properties, "cover-image") ||
			item.ID == coverImageID {
			// re-construct a path, avoid insertion of backslashes as separator on Windows
			coverPath = filepath.ToSlash(filepath.Join(epub.Package[0].BasePath, item.Href))
		}
	}

	// create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	// will close the output file
	defer outputFile.Close()

	// encrypt the content of the publication,
	// write  into the output file
	_, encryptionKey, err := pack.Do(encrypter, contentKey, epub, outputFile)
	if err != nil {
		return err
	}
	pub.EncryptionKey = encryptionKey
	// calculate the output file size and checksum
	stats, err := outputFile.Stat()
	if err == nil && (stats.Size() > 0) {
		filesize := stats.Size()
		pub.Size = uint32(filesize)
		cs := checksum(outputFile)
		pub.Checksum = cs
	}
	if stats.Size() == 0 {
		return errors.New("empty output file")
	}

	if extractCover {
		// extract the cover image and store it at the target location
		for _, f := range zr.File {
			if f.Name == coverPath {
				epubCover, err := f.Open()
				if err != nil {
					log.Printf("Error opening the cover in %s, %s", coverPath, err.Error())
					break // move out of the loop
				}
				defer epubCover.Close()
				// create the output cover
				coverExt := path.Ext(coverPath)
				coverFile, err := os.Create(outputPath + coverExt)
				if err != nil {
					return err
				}
				defer coverFile.Close()
				_, err = io.Copy(coverFile, epubCover)
				if err != nil {
					// we do not consider it as an error
					log.Printf("Error copying cover data, %s", err.Error())
				}
				// set the cover file extension for future storage
				pub.CoverExtension = coverExt
				break
			}
		}
	}

	return nil
}

// processPDF wraps a PDF file inside a Readium Package and encrypts its resources
func processPDF(pub *Publication, inputPath string, outputPath string, encrypter crypto.Encrypter, contentKey string) error {

	log.Println("Process as PDF")

	// generate a temp Readium Package (rwpp) which embeds the PDF file; its title is the PDF file name
	// the first page of the PDF is extracted as a JPEG cover image if extractCover is true
	tmpPackagePath := outputPath + ".tmp"
	coverPath := outputPath + ".jpg"
	rwpInfo, err := pack.BuildRPFFromPDF(inputPath, tmpPackagePath, coverPath)
	// will try to remove the tmp files even if an error is returned
	defer os.Remove(tmpPackagePath)
	// process error
	if err != nil {
		return err
	}

	// set publication metadata extracted from the PDF
	pub.Title = rwpInfo.Title
	pub.Author = rwpInfo.Author
	pub.Subject = rwpInfo.Subject
	pub.CoverExtension = path.Ext(coverPath) // will be ".jpg"

	// build an encrypted package
	return buildEncryptedRPF(pub, tmpPackagePath, outputPath, encrypter, contentKey)
}

// processRPF encrypts the source Readium Package
func processRPF(pub *Publication, inputPath string, outputPath string, encrypter crypto.Encrypter, contentKey string, extractCover bool) error {

	log.Println("Process as RPF")

	// extract the cover from the package if requested
	if extractCover {
		// the cover is copied to coverPath. Its original extension is preserved
		coverPath, err := pack.ExtractCoverFromRPF(inputPath, outputPath)
		// we do not consider err as a fatal error
		if err == nil {
			pub.CoverExtension = path.Ext(coverPath) // will be "" if no cover was found
		}
	}

	// build an encrypted package
	return buildEncryptedRPF(pub, inputPath, outputPath, encrypter, contentKey)
}

// processLPF transforms a W3C LPF file into a Readium Package and encrypts its resources
func processLPF(pub *Publication, inputPath string, outputPath string, encrypter crypto.Encrypter, contentKey string, extractCover bool) error {

	log.Println("Process as LPF")

	// generate a tmp Readium Package (rwpp) out of a W3C Package (lpf)
	tmpPackagePath := outputPath + ".tmp"
	rwpInfo, err := pack.BuildRPFFromLPF(inputPath, tmpPackagePath)
	// will remove the tmp file even if an error is returned
	defer os.Remove(tmpPackagePath)
	// process error
	if err != nil {
		return err
	}

	// set publication metadata
	pub.Title = rwpInfo.Title
	pub.Date = rwpInfo.Date
	pub.Description = rwpInfo.Description
	pub.Language = rwpInfo.Language
	pub.Publisher = rwpInfo.Publisher
	pub.Author = rwpInfo.Author
	pub.Subject = rwpInfo.Subject

	// extract the cover from the package if requested
	if extractCover {
		// the cover is copied to outputPath with its original extension
		coverPath, err := pack.ExtractCoverFromRPF(tmpPackagePath, outputPath)
		// we do not consider err as a fatal error
		if err == nil {
			pub.CoverExtension = path.Ext(coverPath) // will be "" if no cover was found
		}
	}

	// build an encrypted package
	return buildEncryptedRPF(pub, tmpPackagePath, outputPath, encrypter, contentKey)
}

// buildEncryptedRPF builds an encrypted Readium package out of an un-encrypted one
// FIXME: it cannot be used for EPUB as long as Do() and Process() are not merged
func buildEncryptedRPF(pub *Publication, inputPath string, outputPath string, encrypter crypto.Encrypter, contentKey string) error {

	// create a reader on the un-encrypted readium package
	reader, err := pack.OpenRPF(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// create the encrypted package file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	// create a writer on the encrypted package
	writer, err := reader.NewWriter(outputFile)
	if err != nil {
		return err
	}
	// encrypt resources from the input package, return the encryption key
	encryptionKey, err := pack.Process(encrypter, contentKey, reader, writer)
	if err != nil {
		return err
	}
	pub.EncryptionKey = encryptionKey

	err = writer.Close()
	if err != nil {
		return err
	}

	// calculate the output file size and checksum
	stats, err := outputFile.Stat()
	if err == nil && (stats.Size() > 0) {
		filesize := stats.Size()
		pub.Size = uint32(filesize)
		cs := checksum(outputFile)
		pub.Checksum = cs
	}
	if stats.Size() == 0 {
		return errors.New("empty output file")
	}
	return nil
}
