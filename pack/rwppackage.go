// Copyright 2025 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"image/jpeg"
	"path/filepath"
	"strings"
	"time"

	"io"
	"log"
	"net/url"
	"os"

	"github.com/gen2brain/go-fitz"
	"github.com/readium/readium-lcp-server/rwpm"
)

// RWPInfo contains information extracted from a Readium Web Publication,
// deemed useful for a notified CMS or LCP Server
type RWPInfo struct {
	UUID        string
	NumPages    int // only for PDF-based RWPs
	Title       string
	Date        string
	Description string
	Language    []string
	Publisher   []string
	Author      []string
	Subject     []string
}

// RPFReader is a Readium Package reader
type RPFReader struct {
	manifest   rwpm.Publication
	zipArchive *zip.ReadCloser
}

// RPFWriter is a Readium Package writer
type RPFWriter struct {
	manifest  rwpm.Publication
	zipWriter *zip.Writer
}

// NopWriteCloser object
type NopWriteCloser struct {
	io.Writer
}

// NewWriter returns a new PackageWriter writing a RPF file to the output file
func (reader *RPFReader) NewWriter(writer io.Writer) (PackageWriter, error) {

	zipWriter := zip.NewWriter(writer)

	files := map[string]*zip.File{}
	for _, file := range reader.zipArchive.File {
		files[file.Name] = file
	}

	// copy immediately the W3C manifest if it exists in the source package
	if w3cmanFile, ok := files[W3CManifestName]; ok {
		fw, err := zipWriter.Create(W3CManifestName)
		if err != nil {
			return nil, err
		}
		file, err := w3cmanFile.Open()
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, file)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	// copy immediately all ancilliary resources from the source manifest
	// as they will not be encrypted in the current implementation
	// FIXME: work on the encryption of ancilliary resources (except the W3C Entry Page?).
	for _, manifestResource := range reader.manifest.Resources {
		sourceFile := files[manifestResource.Href]
		fw, err := zipWriter.Create(sourceFile.Name)
		if err != nil {
			return nil, err
		}
		file, err := sourceFile.Open()
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, file)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	// copy immediately all linked resources, except the manifest itself (self link),
	// from the source manifest as they should not be encrypted.
	for _, manifestLink := range reader.manifest.Links {
		if manifestLink.Href == ManifestLocation {
			continue
		}
		isSelf := false
		for _, rel := range manifestLink.Rel {
			if rel == "self" {
				isSelf = true
				continue
			}
		}
		if isSelf {
			continue
		}
		sourceFile := files[manifestLink.Href]
		if sourceFile == nil {
			continue
		}
		fw, err := zipWriter.Create(sourceFile.Name)
		if err != nil {
			return nil, err
		}
		file, err := sourceFile.Open()
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, file)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	manifest := reader.manifest

	return &RPFWriter{
		zipWriter: zipWriter,
		manifest:  manifest,
	}, nil
}

// Resources returns a list of all resources which may be encrypted
// It is part of the PackageReader interface.
// Note: the current design choice is to leave ancillary resources (in "resources" and "alternates") unencrypted
// FIXME: add "resources" and "alternates" to the slice
func (reader *RPFReader) Resources() []Resource {
	// index files by name to avoid multiple linear searches
	files := map[string]*zip.File{}
	for _, file := range reader.zipArchive.File {
		files[file.Name] = file
	}

	// list files from the reading order; keep their type and encryption status
	var resources []Resource
	for _, manifestResource := range reader.manifest.ReadingOrder {
		isEncrypted := manifestResource.Properties != nil && manifestResource.Properties.Encrypted != nil
		name, err := url.QueryUnescape(manifestResource.Href)
		if err != nil {
			log.Printf("Error unescaping %s in manifest", manifestResource.Href)
		}
		if files[name] != nil {
			resources = append(resources, &rwpResource{file: files[name], isEncrypted: isEncrypted, contentType: manifestResource.Type})
		} else {
			log.Printf("No file found in the archive for href %s in manifest", manifestResource.Href)
		}
	}

	return resources
}

// ExtractCover extracts the cover image from the Readium Package
func (reader *RPFReader) ExtractCover(coverHref string) (io.Reader, error) {

	// find the cover file in the zip archive
	for _, file := range reader.zipArchive.File {
		if file.Name == coverHref {
			return file.Open()
		}
	}
	return nil, errors.New("no cover file found in the archive")
}

// ExtractCoverHref extracts the cover href from the Readium Package
func (reader *RPFReader) ExtractCoverHref() (string, error) {

	var coverHref string
	// find the cover in the manifest resources
	for _, resource := range reader.manifest.Resources {
		for _, rel := range resource.Rel {
			if rel == "cover" {
				coverHref = resource.Href
				break
			}
		}
		if coverHref != "" {
			break
		}
	}
	if coverHref != "" {
		return coverHref, nil
	}

	// find the cover in the manifest links
	for _, resource := range reader.manifest.Links {
		for _, rel := range resource.Rel {
			if rel == "cover" {
				coverHref = resource.Href
				break
			}
		}
		if coverHref != "" {
			break
		}
	}
	if coverHref != "" {
		return coverHref, nil
	}

	return "", errors.New("no cover found in the manifest")
}

// ConformsTo returns the conformance type of the Readium Package
func (reader *RPFReader) ConformsTo() string {
	return reader.manifest.Metadata.ConformsTo
}

// Title returns the title of the manifest
func (reader *RPFReader) Title() string {
	return reader.manifest.Metadata.Title["und"]
}

// Close closes a Readium Package Reader
func (reader *RPFReader) Close() error {
	return reader.zipArchive.Close()
}

type rwpResource struct {
	isEncrypted bool
	contentType string
	file        *zip.File
}

// rwpResource supports the Resource interface
func (resource *rwpResource) Path() string                   { return resource.file.Name }
func (resource *rwpResource) ContentType() string            { return resource.contentType }
func (resource *rwpResource) Size() int64                    { return int64(resource.file.UncompressedSize64) }
func (resource *rwpResource) Encrypted() bool                { return resource.isEncrypted }
func (resource *rwpResource) Open() (io.ReadCloser, error)   { return resource.file.Open() }
func (resource *rwpResource) CompressBeforeEncryption() bool { return false }
func (resource *rwpResource) CanBeEncrypted() bool           { return true }

// CopyTo copies the resource to the package writer without encryption
func (resource *rwpResource) CopyTo(packageWriter PackageWriter) error {

	wc, err := packageWriter.NewFile(resource.Path(), resource.contentType, resource.file.Method)
	if err != nil {
		return err
	}

	rc, err := resource.file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(wc, rc)

	rCloseError := rc.Close()
	wCloseError := wc.Close()

	if err != nil {
		return err
	}

	if rCloseError != nil {
		return rCloseError
	}

	return wCloseError
}

// Close closes a NopWriteCloser
func (nc *NopWriteCloser) Close() error {
	return nil
}

// NewFile creates a header in the zip archive and adds an entry to the writer reading order if missing.
// This function is called in two main cases:
// - one is the creation of a Readium Package for a PDF file (no existing entry in the manifest)
// - another in the encryption of an existing Readium Package (there is already an entry in the manifest)
// FIXME: the PackageWriter interface is obscure; let's make it better.
func (writer *RPFWriter) NewFile(path string, contentType string, storageMethod uint16) (io.WriteCloser, error) {

	w, err := writer.zipWriter.CreateHeader(&zip.FileHeader{
		Name:   path,
		Method: storageMethod,
	})

	// add an entry to the writer reading order if missing
	found := false
	for _, resource := range writer.manifest.ReadingOrder {
		if path == resource.Href {
			found = true
			break
		}
	}
	if !found {
		writer.manifest.ReadingOrder = append(writer.manifest.ReadingOrder, rwpm.Link{Href: path, Type: contentType})
	}

	return &NopWriteCloser{w}, err
}

// MarkAsEncrypted marks a resource as encrypted (with an algorithm), in the writer manifest
// FIXME: currently only looks into the reading order. Add "alternates", think about adding "resources"
// FIXME: process resources which are compressed before encryption -> add Compression and OriginalLength properties in this case
func (writer *RPFWriter) MarkAsEncrypted(path string, originalSize int64, algorithm string) {

	for i, resource := range writer.manifest.ReadingOrder {
		if path == resource.Href {
			// add encryption properties
			if resource.Properties == nil {
				writer.manifest.ReadingOrder[i].Properties = new(rwpm.Properties)
			}
			writer.manifest.ReadingOrder[i].Properties.Encrypted = &rwpm.Encrypted{
				Scheme: "http://readium.org/2014/01/lcp",
				// profile data is not useful and even misleading: the same encryption algorithm applies to basic and 1.0 profiles.
				//Profile:   profile.String(),
				Algorithm: algorithm,
			}

			break
		}
	}
}

// ManifestLocation is the path if the Readium manifest in a package
const ManifestLocation = "manifest.json"

func (writer *RPFWriter) writeManifest() error {
	w, err := writer.zipWriter.Create(ManifestLocation)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(writer.manifest)
}

// Close closes a Readium Package Writer
// Writes the updated manifest in the zip archive.
func (writer *RPFWriter) Close() error {
	err := writer.writeManifest()
	if err != nil {
		return err
	}

	return writer.zipWriter.Close()
}

// OpenRPF opens a Readium Package and returns a zip reader + a manifest
func OpenRPF(name string) (*RPFReader, error) {

	zipArchive, err := zip.OpenReader(name)
	if err != nil {
		return nil, err
	}

	// find and parse the manifest
	var manifest rwpm.Publication
	var found bool
	for _, file := range zipArchive.File {
		if file.Name == ManifestLocation {
			found = true

			fileReader, err := file.Open()
			if err != nil {
				return nil, err
			}
			decoder := json.NewDecoder(fileReader)

			err = decoder.Decode(&manifest)
			fileReader.Close()
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if !found {
		return nil, errors.New("could not find manifest")
	}

	return &RPFReader{zipArchive: zipArchive, manifest: manifest}, nil
}

// BuildRPFFromPDF builds a Readium Package (rwpp) which embeds a PDF file and a cover
// the cover file extracted from the PDF is not deleted by this function
func BuildRPFFromPDF(inputPath, packagePath, coverPath string, pdfNoMeta bool) (RWPInfo, error) {

	var rwpInfo RWPInfo

	// create the rwpp
	f, err := os.Create(packagePath)
	if err != nil {
		return rwpInfo, err
	}
	defer f.Close()

	// copy the content of the pdf input file into the zip output, as 'publication.pdf'.
	// the pdf content is stored compressed so that the encryption performance on Windows is better (!).
	zipWriter := zip.NewWriter(f)
	defer zipWriter.Close()
	writer, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "publication.pdf",
		Method: zip.Deflate,
	})
	if err != nil {
		return rwpInfo, err
	}
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return rwpInfo, err
	}
	defer inputFile.Close()

	_, err = io.Copy(writer, inputFile)
	if err != nil {
		return rwpInfo, err
	}

	// extract metadata , and cover from the PDF
	rwpInfo, err = extractRWPInfo(inputPath, coverPath)
	if err != nil {
		log.Printf("Error extracting the PDF cover, %s", err.Error())
		return rwpInfo, err
	}

	// add the cover image to the package
	writer, err = zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "cover.jpg",
		Method: zip.Store,
	})
	if err != nil {
		return rwpInfo, err
	}
	coverFile, err := os.Open(coverPath)
	if err != nil {
		return rwpInfo, err
	}
	defer coverFile.Close()

	_, err = io.Copy(writer, coverFile)
	if err != nil {
		return rwpInfo, err
	}

	manifestWriter, err := zipWriter.Create(ManifestLocation)
	if err != nil {
		return rwpInfo, err
	}

	// create simple manifest object
	var manifest rwpm.Publication

	manifest.Context.Add("https://readium.org/webpub-manifest/context.jsonld")
	manifest.Metadata.Type = "http://schema.org/Book"
	manifest.Metadata.ConformsTo = "https://readium.org/webpub-manifest/profiles/pdf"

	// number of pages is needed to display progress in the reader
	manifest.Metadata.NumberOfPages = rwpInfo.NumPages

	// PDF metadata can be so bad that we may want to ignore them
	if pdfNoMeta {
		// we still need a title
		filename := filepath.Base(inputPath)
		rwpInfo.Title = strings.TrimSuffix(filename, filepath.Ext(filename)) // default title
		// remove underscores, hyphens, dots which are frequent in PDF file names
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, "_", " ")
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, "-", " ")
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, ".", " ")
		rwpInfo.Title = strings.TrimSpace(rwpInfo.Title)
		manifest.Metadata.Title.Set("und", rwpInfo.Title)
		// add PDF metadata to the manifest
	} else {
		// remove underscores, hyphens, stars which are frequent in PDF titles
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, "_", " ")
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, "-", " ")
		rwpInfo.Title = strings.ReplaceAll(rwpInfo.Title, "*", " ")
		rwpInfo.Title = strings.TrimSpace(rwpInfo.Title)
		if rwpInfo.Title == "" {
			rwpInfo.Title = "No Title Available" // default title
		}
		manifest.Metadata.Title.Set("und", rwpInfo.Title)
		// there is zero or one author/subject in the PDF metadata
		if len(rwpInfo.Author) != 0 {
			manifest.Metadata.Author.AddName(rwpInfo.Author[0])
		}
		if len(rwpInfo.Subject) != 0 {
			manifest.Metadata.Subject.Add(rwpm.Subject{Name: rwpInfo.Subject[0]})
		}
	}

	manifest.ReadingOrder = []rwpm.Link{
		{
			Href:  "publication.pdf",
			Title: "publication",
			Type:  "application/pdf",
		},
	}
	manifest.Resources = []rwpm.Link{
		{
			Rel:  []string{"cover"},
			Href: "cover.jpg",
			Type: "image/jpeg",
		},
	}

	// marshal and write manifest as JSON
	encoder := json.NewEncoder(manifestWriter)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(manifest)
	if err != nil {
		return rwpInfo, err
	}

	return rwpInfo, nil
}

// extractRWPInfo extracts metadata from the PDF
// and the first page as a JPG image if coverPath is not empty
func extractRWPInfo(inputPath, coverPath string) (RWPInfo, error) {

	var rwpInfo RWPInfo

	// let's check the time it takes to extract the cover
	start := time.Now()
	defer func() {
		if coverPath != "" {
			log.Printf("Extracting the PDF cover took %s", time.Since(start))
		}
	}()

	// we'll use go-fitz to extract the cover and metadata -> CGO-based solution
	doc, err := fitz.New(inputPath)
	if err != nil {
		return rwpInfo, err
	}
	defer doc.Close()

	// extract PDF metadata and number of pages
	metadata := doc.Metadata()
	rwpInfo.Title = cleanNulls(metadata["title"])
	author := cleanNulls(metadata["author"])
	if author != "" {
		rwpInfo.Author = []string{author}
	}
	subject := cleanNulls(metadata["subject"])
	if subject != "" {
		rwpInfo.Subject = []string{subject}
	}
	rwpInfo.NumPages = doc.NumPage()

	if coverPath == "" {
		// no cover extraction requested
		return rwpInfo, nil
	}

	// get the first page
	img, err := doc.Image(0)
	if err != nil {
		return rwpInfo, nil
	}

	// save the image as a JPG file
	cover, err := os.Create(coverPath)
	if err != nil {
		return rwpInfo, err
	}
	defer cover.Close()

	err = jpeg.Encode(cover, img, &jpeg.Options{Quality: jpeg.DefaultQuality})
	if err != nil {
		return rwpInfo, nil
	}

	return rwpInfo, nil
}

// cleanNulls removes null characters from a string
func cleanNulls(s string) string {
	return strings.ReplaceAll(s, string([]byte{0}), "")
}

// ExtractCoverFromRPF extracts the cover image from a Readium Package and saves it to outputRepo
// returns coverPath (with filename and extension) if successful
func ExtractCoverFromRPF(rpfPath, outputRepo string) (string, error) {

	// open the RPF
	rpfReader, err := OpenRPF(rpfPath)
	if err != nil {
		return "", err
	}
	defer rpfReader.Close()

	// extract the cover extension
	coverHref, err := rpfReader.ExtractCoverHref()
	if err != nil {
		return "", err
	}

	// extract the cover image
	coverImage, err := rpfReader.ExtractCover(coverHref)
	if err != nil {
		return "", err
	}

	// build the full path of the cover file
	coverName := filepath.Base(coverHref)
	coverPath := filepath.Join(outputRepo, coverName)

	// save the cover image to the specified path
	coverFile, err := os.Create(coverPath)
	if err != nil {
		return "", err
	}
	defer coverFile.Close()

	_, err = io.Copy(coverFile, coverImage)
	return coverPath, err
}
