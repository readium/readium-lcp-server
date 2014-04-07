package epub

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"sort"
	"strings"

	"github.com/readium/readium-lcp-server/epub/opf"
	"github.com/readium/readium-lcp-server/xmlenc"

	"io"
)

type Epub struct {
	Encryption         *xmlenc.Manifest
	Package            []opf.Package
	Resource           []Resource
	cleartextResources []string
}

func (ep Epub) Cover() (bool, *Resource) {
	for _, p := range ep.Package {
		for _, it := range p.Manifest.Items {
			if strings.Contains(it.Properties, "cover-image") {
				for _, r := range ep.Resource {
					if r.File.Name == filepath.Join(p.BasePath, it.Href) {
						return true, &r
					}
				}
			}
		}
	}

	return false, nil
}

func (ep *Epub) Add(name string, body io.Reader) error {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, body)
	if err != nil {
		return err
	}
	fh := &zip.FileHeader{
		Name:               name,
		UncompressedSize64: uint64(buf.Len()),
		Method:             zip.Deflate,
	}

	ep.Resource = append(ep.Resource, Resource{Output: &buf, FileHeader: fh})

	return nil
}

type Resource struct {
	File       *zip.File
	Output     *bytes.Buffer
	FileHeader *zip.FileHeader
}

func (ep Epub) CanEncrypt(file string) bool {
	i := sort.SearchStrings(ep.cleartextResources, file)
	return i >= len(ep.cleartextResources) || ep.cleartextResources[i] != file
}
