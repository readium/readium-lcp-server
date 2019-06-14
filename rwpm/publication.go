package rwpm

import (
	"errors"
	"path"
	"strings"
)

// Publication Main structure for a publication
type Publication struct {
	Context      []string `json:"@context,omitempty"`
	Metadata     Metadata `json:"metadata"`
	Links        []Link   `json:"links,omitempty:"`
	ReadingOrder []Link   `json:"readingOrder,omitempty"`
	Resources    []Link   `json:"resources,omitempty"` //Replaces the manifest but less redundant
	TOC          []Link   `json:"toc,omitempty"`
	PageList     []Link   `json:"page-list,omitempty"`
	Landmarks    []Link   `json:"landmarks,omitempty"`
	LOI          []Link   `json:"loi,omitempty"` //List of illustrations
	LOA          []Link   `json:"loa,omitempty"` //List of audio files
	LOV          []Link   `json:"lov,omitempty"` //List of videos
	LOT          []Link   `json:"lot,omitempty"` //List of tables

	OtherLinks       []Link                  `json:"-"` //Extension point for links that shouldn't show up in the manifest
	OtherCollections []PublicationCollection `json:"-"` //Extension point for collections that shouldn't show up in the manifest
}

// Link object used in collections and links
type Link struct {
	Href       string      `json:"href"`
	TypeLink   string      `json:"type,omitempty"`
	Rel        []string    `json:"rel,omitempty"`
	Height     int         `json:"height,omitempty"`
	Width      int         `json:"width,omitempty"`
	Title      string      `json:"title,omitempty"`
	Properties *Properties `json:"properties,omitempty"`
	Duration   string      `json:"duration,omitempty"`
	Templated  bool        `json:"templated,omitempty"`
	Children   []Link      `json:"children,omitempty"`
	Bitrate    int         `json:"bitrate,omitempty"`
}

// PublicationCollection is used as an extension points for other collections in a Publication
type PublicationCollection struct {
	Role     string
	Metadata []Meta
	Links    []Link
	Children []PublicationCollection
}

// GetCover return the link for the cover
func (publication *Publication) Cover() (Link, error) {
	return publication.searchLinkByRel("cover")
}

// GetNavDoc return the link for the navigation document
func (publication *Publication) NavDoc() (Link, error) {
	return publication.searchLinkByRel("contents")
}

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

// AddLink Add link in publication link self or search
func (publication *Publication) AddLink(typeLink string, rel []string, url string, templated bool) {
	link := Link{
		Href:     url,
		TypeLink: typeLink,
	}
	if len(rel) > 0 {
		link.Rel = rel
	}

	if templated == true {
		link.Templated = true
	}

	publication.Links = append(publication.Links, link)
}

// AddRel add rel information to Link, will check if the link is already present
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

// AddHrefAbsolute modify Href field with a calculated path based on a
// referend file
func (link *Link) AddHrefAbsolute(href string, baseFile string) {
	link.Href = path.Join(path.Dir(baseFile), href)
}

//TransformLinkToFullURL concatenate a base url to all links
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
