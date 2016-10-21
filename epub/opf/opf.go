// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. 

package opf

import (
	"encoding/xml"
	"io"
)

type Package struct {
	BasePath string   `xml:"-"`
	Metadata Metadata `xml:"http://www.idpf.org/2007/opf metadata"`
	Manifest Manifest `xml:"http://www.idpf.org/2007/opf manifest"`
}
type Metadata struct {
	Author string `json:"author" xml:"http://purl.org/dc/elements/1.1/ creator"`
	Title  string `json:"title" xml:"http://purl.org/dc/elements/1.1/ title"`
	Isbn   string `json:"isbn" xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Metas  []Meta `xml:"http://www.idpf.org/2007/opf meta"`
	Cover  string `json:"cover"`
}

type Manifest struct {
	XMLName xml.Name
	Items   []Item `xml:"http://www.idpf.org/2007/opf item"`
}

func (m Manifest) ItemWithPath(path string) (Item, bool) {
	for _, i := range m.Items {
		if i.Href == path { // FIXME(JPB) Canonicalize the path
			return i, true
		}
	}
	return Item{}, false
}

type Item struct {
	Id         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type Meta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

func findMetas(r io.Reader) ([]Meta, error) {
	var m Metadata
	xd := xml.NewDecoder(r)
	err := xd.Decode(&m)

	return m.Metas, err
}

func Parse(r io.Reader) (Package, error) {
	var p Package
	xd := xml.NewDecoder(r)
	err := xd.Decode(&p)
	return p, err
}
