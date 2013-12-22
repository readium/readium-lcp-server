package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"github.com/jpbougie/lcpserve/epub/opf"
	"github.com/jpbougie/lcpserve/xmlenc"
	"io"
)

const (
	CONTAINER_FILE   = "META-INF/container.xml"
	ENCRYPTION_FILE  = "META-INF/encryption.xml"
	ROOTFILE_ELEMENT = "rootfile"
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
			if start.Name.Local == ROOTFILE_ELEMENT {
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

func Read(r zip.Reader) (Epub, error) {
	container, err := findFileInZip(r, CONTAINER_FILE)
	fd, err := container.Open()
	if err != nil {
		return Epub{}, err
	}
	defer fd.Close()

	rootFiles, err := findRootFiles(fd)
	if err != nil {
		return Epub{}, err
	}

	packages := make([]opf.Package, len(rootFiles))
	for i, rootFile := range rootFiles {
		file, err := findFileInZip(r, rootFile.FullPath)
		if err != nil {
			return Epub{}, err
		}
		packageFile, err := file.Open()
		if err != nil {
			return Epub{}, err
		}
		defer packageFile.Close()

		packages[i], err = opf.Parse(packageFile)
		if err != nil {
			return Epub{}, err
		}
	}

	resources := make([]Resource, 0)

	for _, file := range r.File {
		if file.Name != "META-INF/encryption.xml" &&
			file.Name != "mimetype" {
			resources = append(resources, Resource{File: file, Output: new(bytes.Buffer)})
		}
	}

	var encryption *xmlenc.Manifest
	f, err := findFileInZip(r, ENCRYPTION_FILE)
	if err == nil {
		r, err := f.Open()
		if err != nil {
			return Epub{}, err
		}
		defer r.Close()
		m, err := xmlenc.Read(r)
		encryption = &m
	}

	return Epub{Package: packages, Resource: resources, Encryption: encryption}, nil
}
