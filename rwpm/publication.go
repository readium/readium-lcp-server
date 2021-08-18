// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package rwpm

import (
	"errors"
	"path"
	"strings"
)

// Publication = Readium manifest
type Publication struct {
	Context      MultiString `json:"@context,omitempty"`
	Metadata     Metadata    `json:"metadata"`
	Links        []Link      `json:"links,omitempty"`
	ReadingOrder []Link      `json:"readingOrder,omitempty"`
	Resources    []Link      `json:"resources,omitempty"`
	TOC          []Link      `json:"toc,omitempty"`
	PageList     []Link      `json:"page-list,omitempty"`
	Landmarks    []Link      `json:"landmarks,omitempty"`
	LOI          []Link      `json:"loi,omitempty"` //List of illustrations
	LOA          []Link      `json:"loa,omitempty"` //List of audio files
	LOV          []Link      `json:"lov,omitempty"` //List of videos
	LOT          []Link      `json:"lot,omitempty"` //List of tables

	OtherLinks       []Link                  `json:"-"` //Extension point for links that shouldn't show up in the manifest
	OtherCollections []PublicationCollection `json:"-"` //Extension point for collections that shouldn't show up in the manifest
}

// Link object used in collections and links
type Link struct {
	Href       string      `json:"href"`
	Templated  bool        `json:"templated,omitempty"`
	Type       string      `json:"type,omitempty"`
	Title      string      `json:"title,omitempty"`
	Rel        MultiString `json:"rel,omitempty"`
	Height     int         `json:"height,omitempty"`
	Width      int         `json:"width,omitempty"`
	Duration   float32     `json:"duration,omitempty"`
	Bitrate    int         `json:"bitrate,omitempty"`
	Properties *Properties `json:"properties,omitempty"`
	Alternate  []Link      `json:"alternate,omitempty"`
	Children   []Link      `json:"children,omitempty"`
}

// PublicationCollection is used as an extension point for other collections in a Publication
type PublicationCollection struct {
	Role     string
	Metadata []Meta
	Links    []Link
	Children []PublicationCollection
}

// Cover returns the link relative to the cover
func (publication *Publication) Cover() (Link, error) {
	return publication.searchLinkByRel("cover")
}

// NavDoc returns the link relative to the navigation document
func (publication *Publication) NavDoc() (Link, error) {
	return publication.searchLinkByRel("contents")
}

// SearchLinkByRel returns the link which has a specific relation
func (publication *Publication) searchLinkByRel(rel string) (Link, error) {
	for _, resource := range publication.Resources {
		for _, resRel := range resource.Rel {
			if resRel == rel {
				return resource, nil
			}
		}
	}

	for _, item := range publication.ReadingOrder {
		for _, spineRel := range item.Rel {
			if spineRel == rel {
				return item, nil
			}
		}
	}

	for _, link := range publication.Links {
		for _, linkRel := range link.Rel {
			if linkRel == rel {
				return link, nil
			}
		}
	}

	return Link{}, errors.New("Can't find " + rel + " in publication")
}

// AddLink Adds a link to a publication
func (publication *Publication) AddLink(linkType string, rel []string, url string, templated bool) {
	link := Link{
		Href: url,
		Type: linkType,
	}
	if len(rel) > 0 {
		link.Rel = rel
	}

	if templated == true {
		link.Templated = true
	}

	publication.Links = append(publication.Links, link)
}

// AddRel adds a relation to a Link
func (link *Link) AddRel(rel string) {
	relAlreadyPresent := false

	for _, r := range link.Rel {
		if r == rel {
			relAlreadyPresent = true
			break
		}
	}

	if !relAlreadyPresent {
		link.Rel = append(link.Rel, rel)
	}
}

// AddHrefAbsolute modifies Href with a calculated path based on a referend file
func (link *Link) AddHrefAbsolute(href string, baseFile string) {
	link.Href = path.Join(path.Dir(baseFile), href)
}

// TransformLinkToFullURL adds a base url to every link
func (publication *Publication) TransformLinkToFullURL(baseURL string) {

	for i := range publication.ReadingOrder {
		if !(strings.Contains(publication.ReadingOrder[i].Href, "http://") || strings.Contains(publication.ReadingOrder[i].Href, "https://")) {
			publication.ReadingOrder[i].Href = baseURL + publication.ReadingOrder[i].Href
		}
	}

	for i := range publication.Resources {
		if !(strings.Contains(publication.Resources[i].Href, "http://") || strings.Contains(publication.Resources[i].Href, "https://")) {
			publication.Resources[i].Href = baseURL + publication.Resources[i].Href
		}
	}

	for i := range publication.TOC {
		if !(strings.Contains(publication.TOC[i].Href, "http://") || strings.Contains(publication.TOC[i].Href, "https://")) {
			publication.TOC[i].Href = baseURL + publication.TOC[i].Href
		}
	}

	for i := range publication.Landmarks {
		if !(strings.Contains(publication.Landmarks[i].Href, "http://") || strings.Contains(publication.Landmarks[i].Href, "https://")) {
			publication.Landmarks[i].Href = baseURL + publication.Landmarks[i].Href
		}
	}
}
