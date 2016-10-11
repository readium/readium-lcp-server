package epub

import (
	"archive/zip"
	"path/filepath"
	"sort"
	"strings"
	"io"

	"github.com/readium/readium-lcp-server/epub/opf"
	"github.com/readium/readium-lcp-server/xmlenc"
)

const (
	ContainerFile   = "META-INF/container.xml"
	EncryptionFile  = "META-INF/encryption.xml"
	LicenseFile  = "META-INF/license.lcpl"

	ContentType_XHTML = "application/xhtml+xml"
	ContentType_HTML = "text/html"
	
	ContentType_NCX = "application/x-dtbncx+xml"
	
	ContentType_EPUB = "application/epub+zip"
)

type Epub struct {
	Encryption         *xmlenc.Manifest
	Package            []opf.Package
	Resource           []*Resource
	cleartextResources []string
}

func (ep Epub) Cover() (bool, *Resource) {
	for _, p := range ep.Package {
		for _, it := range p.Manifest.Items {
			if strings.Contains(it.Properties, "cover-image") {
				path := filepath.Join(p.BasePath, it.Href)
				for _, r := range ep.Resource {
					if r.Path == path {
						return true, r
					}
				}
			}
		}
	}

	return false, nil
}

func (ep *Epub) Add(name string, body io.Reader, size uint64) error {
	ep.Resource = append(ep.Resource, &Resource{Contents: body, StorageMethod: zip.Deflate, Path: name, OriginalSize: size})

	return nil
}

type Resource struct {
	Path          string
	ContentType   string
	OriginalSize  uint64
	ContentsSize  uint64
	Compressed    bool
	StorageMethod uint16
	Contents      io.Reader
}

func (ep Epub) CanEncrypt(file string) bool {
	i := sort.SearchStrings(ep.cleartextResources, file)
	return i >= len(ep.cleartextResources) || ep.cleartextResources[i] != file
}
