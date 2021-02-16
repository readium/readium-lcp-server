// Copyright 2019 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package epub

import (
	"archive/zip"
	"io"
	"sort"
	"strings"

	"github.com/endigo/readium-lcp-server/epub/opf"
	"github.com/endigo/readium-lcp-server/xmlenc"
)

const (
	ContainerFile  = "META-INF/container.xml"
	EncryptionFile = "META-INF/encryption.xml"
	LicenseFile    = "META-INF/license.lcpl"

	ContentType_XHTML = "application/xhtml+xml"
	ContentType_HTML  = "text/html"

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

		var coverImageID string
		coverImageID = "cover-image"
		for _, meta := range p.Metadata.Metas {
			if meta.Name == "cover" {
				coverImageID = meta.Content
			}
		}

		for _, it := range p.Manifest.Items {

			if strings.Contains(it.Properties, "cover-image") ||
				it.ID == coverImageID {

				// To be found later, resources in the EPUB root folder
				// must not be prefixed by "./"
				path := it.Href
				if p.BasePath != "." {
					path = p.BasePath + "/" + it.Href
				}
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
