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
	UUID          string
	AltID         string
	Title         string
	Date          string
	Description   string
	Language      []string
	Publisher     []string
	Author        []string
	Subject       []string
	InputPath     string
	ExtractCover  bool
	CoverUrl      string
	StorageMode   int
	OutputRepo    string
	FileName      string
	CoverName     string
	EncryptionKey []byte
	Location      string
	ContentType   string
	Size          uint32
	Checksum      string
}

// ProcessEncryption encrypts a publication
// inputPath must contain a processable file extension.
func ProcessEncryption(contentID, contentKey, inputPath, tempRepo, outputRepo, storageRepo, storageURL, storageFilename string, extractCover, pdfNoMeta bool) (*Publication, error) {

	if inputPath == "" {
		return nil, errors.New("ProcessEncryption, missing input path")
	}
	log.Println("Process ", inputPath)

	var pub Publication
	pub.OutputRepo = outputRepo
	pub.ExtractCover = extractCover
	pub.InputPath = inputPath
	// set the AltID as the file name without extension
	pub.AltID = strings.TrimSuffix(filepath.Base(pub.InputPath), filepath.Ext(pub.InputPath))

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
		pub.InputPath = tempPath
	}

	// select a storage mode
	pub.StorageMode = apilcp.Storage_none
	// if the storage repo is set, set storage mode and output repository
	// note: the -storage parameter takes precedence over -output
	if storageRepo != "" {
		// S3 storage is specified by the presence of "s3:" at the start of the -storage param
		if strings.HasPrefix(storageRepo, "s3:") {
			pub.StorageMode = apilcp.Storage_s3
			pub.OutputRepo = tempRepo // before move to s3
		} else {
			// file system storage
			pub.StorageMode = apilcp.Storage_fs
			// create the storage folder when necessary
			err := os.MkdirAll(storageRepo, os.ModePerm)
			if err != nil && !os.IsExist(err) {
				return nil, err
			}
			// the encrypted file will be directly generated inside the storage path
			pub.OutputRepo = storageRepo
		}
	}
	// if the output repo is still not set, use the temp directory.
	if pub.OutputRepo == "" {
		pub.OutputRepo = tempRepo
	}

	// set the target file name and content type
	err = setTargetFileInfo(&pub, storageFilename)
	if err != nil {
		return nil, err
	}

	// define an AES encrypter
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// the cover should only be extracted if lcpencrypt stores the file (not if the LCP Server stores the file)
	if storageRepo == "" {
		pub.ExtractCover = false
	}

	// select the encryption process from the input file extension
	inputExt := filepath.Ext(inputPath)
	switch inputExt {
	case ".epub":
		err = processEPUB(&pub, encrypter, contentKey)
	case ".pdf":
		err = processPDF(&pub, encrypter, contentKey, pdfNoMeta)
	case ".lpf":
		err = processLPF(&pub, encrypter, contentKey)
	case ".audiobook", ".divina", ".webpub", ".rpf":
		err = processRPF(&pub, encrypter, contentKey)
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
		pub.Location = filepath.Join(pub.OutputRepo, pub.FileName)
		mode = "temp"
	case apilcp.Storage_fs:
		// the encryption tool stores the encrypted publication in a file system
		// location indicates the url of the publication
		pub.Location, err = url.JoinPath(storageURL, pub.FileName)
		if err != nil {
			return nil, err
		}
		mode = "file system"
	case apilcp.Storage_s3:
		// the encryption tool stores the encrypted publication in an S3 storage
		// and delete the temp file
		fromPath := filepath.Join(pub.OutputRepo, pub.FileName)
		err = StoreFileOnS3(fromPath, storageRepo, pub.FileName)
		if err != nil {
			return nil, err
		}
		// if a cover was extracted (pub.CoverName not empty), store it in S3 too
		// and delete the cover
		if pub.ExtractCover && pub.CoverName != "" {
			fromPath := filepath.Join(pub.OutputRepo, pub.CoverName)
			err = StoreFileOnS3(fromPath, storageRepo, pub.CoverName)
			if err != nil {
				return nil, err
			}
		}

		// location indicates the url of the publication on S3
		pub.Location, err = url.JoinPath(storageURL, pub.FileName)
		if err != nil {
			return nil, err
		}
		mode = "s3"
	}
	log.Println("Storage mode", mode)

	// if a cover was extracted, set its url
	if pub.ExtractCover && pub.CoverName != "" {
		pub.CoverUrl, _ = url.JoinPath(storageURL, pub.CoverName)
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

// setTargetFileInfo sets the file name and content type
// which will be used during future downloads
func setTargetFileInfo(pub *Publication, storageFilename string) error {

	var targetExt string
	switch filepath.Ext(pub.InputPath) {
	case ".epub":
		targetExt = ".epub"
		// epub is a special case, as the content type does not change for encrypted files
		pub.ContentType = epub.ContentType_EPUB
	case ".pdf":
		targetExt = ".lcpdf"
		pub.ContentType = "application/pdf+lcp"
	case ".audiobook":
		targetExt = ".lcpa"
		pub.ContentType = "application/audiobook+lcp"
	case ".divina":
		targetExt = ".lcpdi"
		pub.ContentType = "application/divina+lcp"
	case ".webpub", ".rpf", ".lpf":
		targetExt = ".webpub"
		// Temporary value. The conformsTo property of the manifest will be checked later
		pub.ContentType = "application/webpub+lcp"
	default:
		return errors.New("unprocessable extension " + filepath.Ext(pub.InputPath))
	}

	// if the storage filename is imposed, use it
	if storageFilename != "" {
		// remove the extension (if any)
		storageFilename = strings.TrimSuffix(storageFilename, filepath.Ext(storageFilename))
	} else {
		// use the content ID as file name, add the target extension
		storageFilename = pub.UUID
	}

	// set the publication filename with the target extension
	pub.FileName = storageFilename + targetExt

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
func processEPUB(pub *Publication, encrypter crypto.Encrypter, contentKey string) error {

	log.Println("Process as EPUB")

	// create a zip reader from the input path
	zr, err := zip.OpenReader(pub.InputPath)
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

	// create the output file
	outputFile, err := os.Create(filepath.Join(pub.OutputRepo, pub.FileName))
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

	// look for the cover image, in case its extraction is requested
	var coverImageID string
	var coverPath string
	if pub.ExtractCover {
		for _, meta := range epub.Package[0].Metadata.Metas {
			if meta.Name == "cover" {
				coverImageID = meta.Content
			}
		}
		for _, item := range epub.Package[0].Manifest.Items {
			if strings.Contains(item.Properties, "cover-image") ||
				item.ID == coverImageID {
				// re-construct a path, avoid insertion of backslashes as separator on Windows
				coverPath = filepath.ToSlash(filepath.Join(epub.Package[0].BasePath, item.Href))
			}
		}
		// extract the cover image and store it at the target location
		for _, f := range zr.File {
			if f.Name == coverPath {
				epubCover, err := f.Open()
				if err != nil {
					log.Printf("Error opening the cover in %s, %s", coverPath, err.Error())
					break // move out of the loop
				}
				defer epubCover.Close()
				// create the output cover file
				pub.CoverName = strings.TrimSuffix(pub.FileName, filepath.Ext(pub.FileName)) + filepath.Ext(coverPath)
				coverFile, err := os.Create(filepath.Join(pub.OutputRepo, pub.CoverName))
				if err != nil {
					return err
				}
				defer coverFile.Close()
				_, err = io.Copy(coverFile, epubCover)
				if err != nil {
					// we do not consider it as a fatal error
					log.Printf("Error copying cover data, %s", err.Error())
				}
				// we found the cover, exit the loop
				break
			}
		}
	}
	return nil
}

// processPDF wraps a PDF file inside a Readium Package and encrypts its resources
func processPDF(pub *Publication, encrypter crypto.Encrypter, contentKey string, pdfNoMeta bool) error {

	log.Println("Process as PDF")

	tmpPackagePath := filepath.Join(pub.OutputRepo, pub.FileName+".tmp")
	pub.CoverName = strings.TrimSuffix(pub.FileName, filepath.Ext(pub.FileName)) + ".jpg"
	coverPath := filepath.Join(pub.OutputRepo, pub.CoverName)

	// generate a temp Readium Package (rwpp) which embeds the PDF file
	// the first page of the PDF is extracted as a JPEG cover image
	rwpInfo, err := pack.BuildRPFFromPDF(pub.InputPath, tmpPackagePath, coverPath, pdfNoMeta)
	// will will remove the tmp file even if an error is returned
	defer os.Remove(tmpPackagePath)
	// process error
	if err != nil {
		return err
	}

	if !pub.ExtractCover {
		// remove the cover file if it was created
		os.Remove(coverPath)
		pub.CoverName = ""
	}

	// set publication metadata extracted from the PDF
	pub.Title = rwpInfo.Title
	pub.Author = rwpInfo.Author
	pub.Subject = rwpInfo.Subject

	// build an encrypted package
	pub.InputPath = tmpPackagePath
	return buildEncryptedRPF(pub, encrypter, contentKey)
}

// processRPF encrypts the source Readium Package
func processRPF(pub *Publication, encrypter crypto.Encrypter, contentKey string) error {

	log.Println("Process as Readium Package")

	// extract the cover from the package if requested
	if pub.ExtractCover {
		// the cover is copied to coverPath. Its original extension is preserved
		coverPath, err := pack.ExtractCoverFromRPF(pub.InputPath, pub.OutputRepo)
		// we do not consider err as a fatal error
		if err != nil {
			log.Println("No cover extracted from the RPF. Error:", err.Error())
		} else {
			pub.CoverName = filepath.Base(coverPath) // will be "" if no cover was found
		}
	}

	// build an encrypted package
	return buildEncryptedRPF(pub, encrypter, contentKey)
}

// processLPF transforms a W3C LPF file into a Readium Package and encrypts its resources
func processLPF(pub *Publication, encrypter crypto.Encrypter, contentKey string) error {

	log.Println("Process as W3C LPF Package")

	// generate a tmp Readium Package (rwpp) out of a W3C Package (lpf)
	tmpPackagePath := filepath.Join(pub.OutputRepo, pub.FileName+".tmp")
	rwpInfo, err := pack.BuildRPFFromLPF(pub.InputPath, tmpPackagePath)
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
	if pub.ExtractCover {
		// the cover is copied to outputRepo. Its original extension is preserved
		coverPath, err := pack.ExtractCoverFromRPF(tmpPackagePath, pub.OutputRepo)
		// we do not consider err as a fatal error
		if err != nil {
			log.Println("No cover extracted from the LPF. Error:", err.Error())
		} else {
			pub.CoverName = filepath.Base(coverPath)
		}
	}

	// build an encrypted package from a new input file
	pub.InputPath = tmpPackagePath
	return buildEncryptedRPF(pub, encrypter, contentKey)
}

// buildEncryptedRPF builds an encrypted Readium package out of an un-encrypted one
func buildEncryptedRPF(pub *Publication, encrypter crypto.Encrypter, contentKey string) error {

	// create a reader on the un-encrypted readium package
	reader, err := pack.OpenRPF(pub.InputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// set the title from the manifest if not already set
	if pub.Title == "" {
		pub.Title = reader.Title()
	}

	// set the target content type from the conformance type in the manifest
	ext := filepath.Ext(pub.FileName)
	switch reader.ConformsTo() {
	case "https://readium.org/webpub-manifest/profiles/audiobook":
		pub.FileName = strings.TrimSuffix(pub.FileName, ext) + ".lcpa"
		pub.ContentType = "application/audiobook+lcp"
	case "https://readium.org/webpub-manifest/profiles/divina":
		pub.FileName = strings.TrimSuffix(pub.FileName, ext) + ".lcpdi"
		pub.ContentType = "application/divina+lcp"
	case "https://readium.org/webpub-manifest/profiles/pdf":
		pub.FileName = strings.TrimSuffix(pub.FileName, ext) + ".lcpdf"
		pub.ContentType = "application/pdf+lcp"
	}

	// create the encrypted package file
	outputFile, err := os.Create(filepath.Join(pub.OutputRepo, pub.FileName))
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
