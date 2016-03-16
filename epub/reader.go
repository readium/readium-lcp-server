package epub

import (
	"archive/zip"
	"encoding/xml"
	"path/filepath"

	"github.com/readium/readium-lcp-server/epub/opf"
	"github.com/readium/readium-lcp-server/xmlenc"

	"io"
	"sort"
	"strings"
)

const (
	containerFile  = "META-INF/container.xml"
	encryptionFile = "META-INF/encryption.xml"
)

type rootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

func findRootFiles(r io.Reader) ([]rootFile, error) {
	xd := xml.NewDecoder(r)
	var roots []rootFile
	for x, err := xd.Token(); x != nil && err == nil; x, err = xd.Token() {
		if err != nil {
			return nil, err
		}
		switch x.(type) {
		case xml.StartElement:
			start := x.(xml.StartElement)
			if start.Name.Local == "rootfile" {
				var file rootFile
				err = xd.DecodeElement(&file, &start)
				if err != nil {
					return nil, err
				}
				roots = append(roots, file)
			}
		}
	}

	return roots, nil
}

func (ep *Epub) addCleartextResources(names []string) {
	if ep.cleartextResources == nil {
		ep.cleartextResources = []string{}
	}

	for _, name := range names {
		ep.cleartextResources = append(ep.cleartextResources, name)
	}
}

func (ep *Epub) addCleartextResource(name string) {
	if ep.cleartextResources == nil {
		ep.cleartextResources = []string{}
	}

	ep.cleartextResources = append(ep.cleartextResources, name)
}

func Read(r *zip.Reader) (Epub, error) {
	var ep Epub
	container, err := findFileInZip(r, containerFile)
	if err != nil {
		return ep, err
	}
	fd, err := container.Open()
	if err != nil {
		return ep, err
	}
	defer fd.Close()

	rootFiles, err := findRootFiles(fd)
	if err != nil {
		return ep, err
	}

	packages := make([]opf.Package, len(rootFiles))
	for i, rootFile := range rootFiles {
		ep.addCleartextResource(rootFile.FullPath)
		file, err := findFileInZip(r, rootFile.FullPath)
		if err != nil {
			return ep, err
		}
		packageFile, err := file.Open()
		if err != nil {
			return ep, err
		}
		defer packageFile.Close()

		packages[i], err = opf.Parse(packageFile)
		packages[i].BasePath = filepath.Dir(rootFile.FullPath)
		addCleartextResources(&ep, packages[i])
		if err != nil {
			return ep, err
		}
	}

	var resources []*Resource

	for _, file := range r.File {
		if file.Name != "META-INF/encryption.xml" &&
			file.Name != "mimetype" {
			in, err := file.Open()
			if err != nil {
				return ep, err
			}
			resource := &Resource{Path: file.Name, Contents: in, OriginalSize: file.FileHeader.UncompressedSize64}
			if item, ok := findResourceInPackages(resource, packages); ok {
				resource.ContentType = item.MediaType
			}
			resources = append(resources, resource)
		}
		if strings.HasPrefix(file.Name, "META-INF") {
			ep.addCleartextResource(file.Name)
		}
	}

	var encryption *xmlenc.Manifest
	f, err := findFileInZip(r, encryptionFile)
	if err == nil {
		r, err := f.Open()
		if err != nil {
			return Epub{}, err
		}
		defer r.Close()
		m, err := xmlenc.Read(r)
		encryption = &m
	}

	ep.Package = packages
	ep.Resource = resources
	ep.Encryption = encryption
	sort.Strings(ep.cleartextResources)

	return ep, nil
}

func addCleartextResources(ep *Epub, p opf.Package) {
	// Look for cover, nav and NCX items
	for _, item := range p.Manifest.Items {
		if strings.Contains(item.Properties, "cover-image") ||
			strings.Contains(item.Properties, "nav") ||
			item.MediaType == "application/x-dtbncx+xml" {
			ep.addCleartextResource(filepath.Join(p.BasePath, item.Href))
		}
	}
}

func findResourceInPackages(r *Resource, packages []opf.Package) (opf.Item, bool) {
	for _, p := range packages {
		relative, err := filepath.Rel(p.BasePath, r.Path)
		if err != nil {
			return opf.Item{}, false
		}

		if item, ok := p.Manifest.ItemWithPath(relative); ok {
			return item, ok
		}
	}

	return opf.Item{}, false
}
