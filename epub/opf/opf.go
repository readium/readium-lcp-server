// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package opf

import (
	"encoding/xml"
	"io"

	"golang.org/x/net/html/charset"
)

// Package is the main opf structure
type Package struct {
	BasePath string   `xml:"-"`
	Metadata Metadata `xml:"http://www.idpf.org/2007/opf metadata"`
	Manifest Manifest `xml:"http://www.idpf.org/2007/opf manifest"`
}

// Metadata is the package metadata structure
type Metadata struct {
	Author string `json:"author" xml:"http://purl.org/dc/elements/1.1/ creator"`
	Title  string `json:"title" xml:"http://purl.org/dc/elements/1.1/ title"`
	Isbn   string `json:"isbn" xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Metas  []Meta `xml:"http://www.idpf.org/2007/opf meta"`
	Cover  string `json:"cover"`
}

// Meta is the metadata item structure
type Meta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// Manifest is the package manifest structure
type Manifest struct {
	XMLName xml.Name
	Items   []Item `xml:"http://www.idpf.org/2007/opf item"`
}

// Item is the manifest item structure
type Item struct {
	Id         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// ItemWithPath looks for the manifest item corresponding to a given path
func (m Manifest) ItemWithPath(path string) (Item, bool) {
	for _, i := range m.Items {
		if i.Href == path { // FIXME(JPB) Canonicalize the path
			return i, true
		}
	}
	return Item{}, false
}

// Parse parses the opf xml struct and returns a Package object
func Parse(r io.Reader) (Package, error) {
	var p Package
	xd := xml.NewDecoder(r)
	// deal with non utf-8 xml files
	xd.CharsetReader = charset.NewReaderLabel
	err := xd.Decode(&p)
	return p, err
}
