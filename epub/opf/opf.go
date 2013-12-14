package opf

import (
  "encoding/xml"
  "io"
)

type Package struct {
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

type Item struct {
	Id        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
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
