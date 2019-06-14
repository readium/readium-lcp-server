package pack

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"text/template"

	"github.com/readium/readium-lcp-server/rwpm"
)

type RWPPackageReader struct {
	manifest   rwpm.Publication
	zipArchive *zip.Reader
}

func (reader *RWPPackageReader) NewWriter(writer io.Writer) (PackageWriter, error) {
	zipWriter := zip.NewWriter(writer)

	files := map[string]*zip.File{}
	for _, file := range reader.zipArchive.File {
		files[file.Name] = file
	}

	// copy all ancilliary resources for now as they should not be encrypted
	for _, manifestResource := range reader.manifest.Resources {
		sourceFile := files[manifestResource.Href]
		fw, err := zipWriter.Create(sourceFile.Name)
		if err != nil {
			return nil, err
		}
		file, err := sourceFile.Open()
		_, err = io.Copy(fw, file)
		file.Close()
	}

	manifest := reader.manifest
	manifest.ReadingOrder = nil

	return &RWPPackageWriter{
		zipWriter: zipWriter,
		manifest:  manifest,
	}, nil
}

func (reader *RWPPackageReader) Resources() []Resource {
	// Index files by name to avoid multiple linear searches
	files := map[string]*zip.File{}
	for _, file := range reader.zipArchive.File {
		files[file.Name] = file
	}

	// Content items from the spine, that could be encrypted
	var resources []Resource
	for _, manifestResource := range reader.manifest.ReadingOrder {
		isEncrypted := manifestResource.Properties != nil && manifestResource.Properties.Encrypted != nil
		resources = append(resources, &rwpResource{file: files[manifestResource.Href], isEncrypted: isEncrypted, contentType: manifestResource.TypeLink})
	}

	return resources
}

type rwpResource struct {
	isEncrypted bool
	contentType string
	file        *zip.File
}

func (resource *rwpResource) Path() string                   { return resource.file.Name }
func (resource *rwpResource) ContentType() string            { return resource.contentType }
func (resource *rwpResource) Size() int64                    { return int64(resource.file.UncompressedSize64) }
func (resource *rwpResource) Encrypted() bool                { return resource.isEncrypted }
func (resource *rwpResource) Open() (io.ReadCloser, error)   { return resource.file.Open() }
func (resource *rwpResource) CompressBeforeEncryption() bool { return false }
func (resource *rwpResource) CanBeEncrypted() bool           { return true }
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

type RWPPackageWriter struct {
	manifest  rwpm.Publication
	zipWriter *zip.Writer
}

type NopWriteCloser struct {
	io.Writer
}

func (nc *NopWriteCloser) Close() error {
	return nil
}

func (writer *RWPPackageWriter) NewFile(path string, contentType string, storageMethod uint16) (io.WriteCloser, error) {
	w, err := writer.zipWriter.CreateHeader(&zip.FileHeader{
		Name:   path,
		Method: storageMethod,
	})

	writer.manifest.ReadingOrder = append(writer.manifest.ReadingOrder, rwpm.Link{
		Href:     path,
		TypeLink: contentType,
	})

	return &NopWriteCloser{w}, err
}

func (writer *RWPPackageWriter) MarkAsEncrypted(path string, originalSize int64, profile EncryptionProfile, algorithm string) {
	for i, resource := range writer.manifest.ReadingOrder {
		if path == resource.Href {
			if resource.Properties == nil {
				writer.manifest.ReadingOrder[i].Properties = new(rwpm.Properties)
			}

			writer.manifest.ReadingOrder[i].Properties.Encrypted = &rwpm.Encrypted{
				Scheme:    "http://readium.org/2014/01/lcp",
				Profile:   string(profile),
				Algorithm: algorithm,
			}

			break
		}
	}
}

const MANIFEST_LOCATION = "manifest.json"

func (writer *RWPPackageWriter) writeManifest() error {
	w, err := writer.zipWriter.Create(MANIFEST_LOCATION)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(writer.manifest)
}

func (writer *RWPPackageWriter) Close() error {
	err := writer.writeManifest()
	if err != nil {
		return err
	}

	return writer.zipWriter.Close()
}

func NewPackagedRWPReader(zipReader *zip.Reader) (*RWPPackageReader, error) {
	// Find and parse the manifest
	var manifest rwpm.Publication
	var found bool
	for _, file := range zipReader.File {
		if file.Name == MANIFEST_LOCATION {
			found = true

			readCloser, err := file.Open()
			if err != nil {
				return nil, err
			}
			decoder := json.NewDecoder(readCloser)

			err = decoder.Decode(&manifest)
			readCloser.Close()
			if err != nil {
				return nil, err
			}

			break
		}
	}

	if !found {
		return nil, errors.New("Could not find manifest")
	}

	return &RWPPackageReader{zipArchive: zipReader, manifest: manifest}, nil

}

func OpenPackagedRWP(name string) (*RWPPackageReader, error) {
	zipArchive, err := zip.OpenReader(name)
	if err != nil {
		return nil, err
	}

	return NewPackagedRWPReader(&zipArchive.Reader)
}

func BuildWebPubPackageFromPDF(title string, inputPath string, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	defer f.Close()

	zipWriter := zip.NewWriter(f)
	writer, err := zipWriter.Create("publication.pdf")
	if err != nil {
		return err
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		zipWriter.Close()
		return err
	}
	defer inputFile.Close()

	_, err = io.Copy(writer, inputFile)
	if err != nil {
		zipWriter.Close()
		return err
	}

	manifest := `
	{
		"@context": [
			"https://readium.org/webpub-manifest/context.jsonld",
			"https://readium.org/webpub-manifest/contexts/epub/context.jsonld"
		],
		"metadata": {
			"title": "{{.Title}}"
		},
		"readingOrder": [
			{
				"href": "publication.pdf",
				"type": "application/pdf"
			}
		]
	}
	`

	manifestWriter, err := zipWriter.Create("manifest.json")
	if err != nil {
		return err
	}

	tmpl, err := template.New("manifest").Parse(manifest)
	if err != nil {
		zipWriter.Close()
		return err
	}

	err = tmpl.Execute(manifestWriter, struct{ Title string }{title})
	if err != nil {
		zipWriter.Close()
		return err
	}

	return zipWriter.Close()
}
