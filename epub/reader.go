// Copyright 2019 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/readium/readium-lcp-server/epub/opf"
	"github.com/readium/readium-lcp-server/xmlenc"
	"golang.org/x/net/html/charset"
)

// root element of the opf
const (
	RootFileElement = "rootfile"
)

type rootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// findRootFiles looks for the epub root files
func findRootFiles(r io.Reader) ([]rootFile, error) {
	xd := xml.NewDecoder(r)
	// deal with non utf-8 xml files
	xd.CharsetReader = charset.NewReaderLabel
	var roots []rootFile
	for x, err := xd.Token(); x != nil && err == nil; x, err = xd.Token() {
		switch x.(type) {
		case xml.StartElement:
			start := x.(xml.StartElement)
			if start.Name.Local == RootFileElement {
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

func (ep *Epub) addCleartextResource(name string) {
	if ep.cleartextResources == nil {
		ep.cleartextResources = []string{}
	}

	ep.cleartextResources = append(ep.cleartextResources, name)
}

// Read reads the opf file in the zip passed as a parameter,
// selects resources which mustn't be encrypted
// and returns an EPUB object
func Read(r *zip.Reader) (Epub, error) {
	var ep Epub
	container, err := findFileInZip(r, ContainerFile)
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
		if err != nil {
			fmt.Println("Error parsing the opf file")
			return ep, err
		}
		packages[i].BasePath = filepath.Dir(rootFile.FullPath)
		addCleartextResources(&ep, packages[i])
	}

	var resources []*Resource

	var encryption *xmlenc.Manifest
	f, err := findFileInZip(r, EncryptionFile)
	if err == nil {
		r, err := f.Open()
		if err != nil {
			return Epub{}, err
		}
		defer r.Close()
		m, _ := xmlenc.Read(r)
		encryption = &m
	}

	for _, file := range r.File {

		// EPUBs do not require us to keep directory entries and we cannot process them
		if file.FileInfo().IsDir() {
			continue
		}

		if file.Name != EncryptionFile &&
			file.Name != "mimetype" {
			rc, err := file.Open()
			if err != nil {
				return Epub{}, err
			}
			compressed := false

			if encryption != nil {
				if data, ok := encryption.DataForFile(file.Name); ok {
					if data.Properties != nil {
						for _, prop := range data.Properties.Properties {
							if prop.Compression.Method == 8 {
								compressed = true
								break
							}
						}
					}
				}
			}

			resource := &Resource{Path: file.Name, Contents: rc, StorageMethod: file.Method, OriginalSize: file.FileHeader.UncompressedSize64, Compressed: compressed}
			if item, ok := findResourceInPackages(resource, packages); ok {
				resource.ContentType = item.MediaType
			}
			resources = append(resources, resource)
		}
		if strings.HasPrefix(file.Name, "META-INF") {
			ep.addCleartextResource(file.Name)
		}
	}

	ep.Package = packages
	ep.Resource = resources
	ep.Encryption = encryption
	sort.Strings(ep.cleartextResources)

	return ep, nil
}

// addCleartextResources searches for resources which must no be encrypted
// i.e. cover, nav and NCX
func addCleartextResources(ep *Epub, p opf.Package) {
	coverImageID := "cover-image"
	for _, meta := range p.Metadata.Metas {
		if meta.Name == "cover" {
			coverImageID = meta.Content
		}
	}

	// Look for cover, nav and NCX items
	for _, item := range p.Manifest.Items {
		if strings.Contains(item.Properties, "cover-image") ||
			item.ID == coverImageID ||
			strings.Contains(item.Properties, "nav") ||
			item.MediaType == ContentType_NCX {
			// re-construct a path, avoid insertion of backslashes as separator on Windows
			path := filepath.ToSlash(filepath.Join(p.BasePath, item.Href))
			ep.addCleartextResource(path)
		}
	}
}

// findResourceInPackages returns an opf item which corresponds to
// the path of the resource given as parameter
func findResourceInPackages(r *Resource, packages []opf.Package) (opf.Item, bool) {
	for _, p := range packages {
		relative, err := filepath.Rel(p.BasePath, r.Path)
		if err != nil {
			return opf.Item{}, false
		}
		// avoid insertion of backslashes as separator on Windows
		relative = filepath.ToSlash(relative)

		if item, ok := p.Manifest.ItemWithPath(relative); ok {
			return item, ok
		}
	}

	return opf.Item{}, false
}
