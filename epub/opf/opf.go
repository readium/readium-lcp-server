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
	Identifier  string   `json:"identifier" xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Title       []string `json:"title" xml:"http://purl.org/dc/elements/1.1/ title"`
	Description string   `json:"description" xml:"http://purl.org/dc/elements/1.1/ description"`
	Date        string   `json:"date" xml:"http://purl.org/dc/elements/1.1/ date"`
	Author      []string `json:"author" xml:"http://purl.org/dc/elements/1.1/ creator"`
	Contributor []string `json:"contributor" xml:"http://purl.org/dc/elements/1.1/ contributor"`
	Publisher   []string `json:"publisher" xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Language    []string `json:"language" xml:"http://purl.org/dc/elements/1.1/ language"`
	Subject     []string `json:"subject" xml:"http://purl.org/dc/elements/1.1/ subject"`
	Metas       []Meta   `xml:"http://www.idpf.org/2007/opf meta"`
}

// Meta is the metadata item structure
type Meta struct {
	Name     string `xml:"name,attr"` // EPUB 2
	Content  string `xml:"content,attr"`
	Property string `xml:"property,attr"` // EPUB 3
	Refines  string `xml:"refines,attr"`
	Text     string `xml:",chardata"`
}

// Manifest is the package manifest structure
type Manifest struct {
	Items []Item `xml:"http://www.idpf.org/2007/opf item"`
}

// Item is the manifest item structure
type Item struct {
	ID         string `xml:"id,attr"`
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
