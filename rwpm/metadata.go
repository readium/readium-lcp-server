package rwpm

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

// Metadata for the default context in WebPub
type Metadata struct {
	RDFType         string        `json:"@type,omitempty"` //Defaults to schema.org for EBook
	Title           MultiLanguage `json:"title"`
	Identifier      string        `json:"identifier,omitempty"`
	Author          Contributors  `json:"author,omitempty"`
	Translator      Contributors  `json:"translator,omitempty"`
	Editor          Contributors  `json:"editor,omitempty"`
	Artist          Contributors  `json:"artist,omitempty"`
	Illustrator     Contributors  `json:"illustrator,omitempty"`
	Letterer        Contributors  `json:"letterer,omitempty"`
	Penciler        Contributors  `json:"penciler,omitempty"`
	Colorist        Contributors  `json:"colorist,omitempty"`
	Inker           Contributors  `json:"inker,omitempty"`
	Narrator        Contributors  `json:"narrator,omitempty"`
	Contributor     Contributors  `json:"contributor,omitempty"`
	Publisher       Contributors  `json:"publisher,omitempty"`
	Imprint         Contributors  `json:"imprint,omitempty"`
	Language        []string      `json:"language,omitempty"`
	Modified        *time.Time    `json:"modified,omitempty"`
	PublicationDate *time.Time    `json:"published,omitempty"`
	Description     string        `json:"description,omitempty"`
	Direction       string        `json:"direction,omitempty"`
	Rendition       *Properties   `json:"rendition,omitempty"`
	Source          string        `json:"source,omitempty"`
	EpubType        []string      `json:"epub-type,omitempty"`
	Rights          string        `json:"rights,omitempty"`
	Subject         []Subject     `json:"subject,omitempty"`
	BelongsTo       *BelongsTo    `json:"belongs_to,omitempty"`
	Duration        int           `json:"duration,omitempty"`

	OtherMetadata []Meta `json:"-"` //Extension point for other metadata
}

// Meta is a generic structure for other metadata
type Meta struct {
	Property string
	Value    interface{}
	Children []Meta
}

type Contributors []Contributor

func (c *Contributors) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	if len(b) == 0 {
		return errors.New("Cannot parse a 0-length buffer")
	}

	if b[0] == '"' || b[0] == '{' {
		var contributor Contributor
		err := contributor.UnmarshalJSON(b)
		if err != nil {
			return err
		}
		*c = []Contributor{contributor}
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	_, err := decoder.Token() // [
	var ctors []Contributor
	for {
		if !decoder.More() {
			break
		}

		var contributor Contributor

		err = decoder.Decode(&contributor)
		if err != nil {
			return err
		}

		ctors = append(ctors, contributor)
		*c = ctors

	}

	return nil
}

func (c Contributors) MarshalJSON() ([]byte, error) {
	if len(c) == 1 {
		return json.Marshal(c[0])
	}

	var buf bytes.Buffer
	for i, contributor := range c {
		if i != 0 {
			buf.WriteRune(',')
		}
		b, err := json.Marshal(contributor)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	return buf.Bytes(), nil
}

// Contributor construct used internally for all contributors
type Contributor struct {
	Name       MultiLanguage `json:"name,omitempty"`
	SortAs     string        `json:"sort_as,omitempty"`
	Identifier string        `json:"identifier,omitempty"`
	Role       string        `json:"role,omitempty"`
}

func (c *Contributor) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	t, err := dec.Token()
	if err != nil {
		return err
	}

	if s, ok := t.(string); ok {
		c.Name = MultiLanguage{SingleString: s}
		return nil
	}

	if d, ok := t.(json.Delim); !ok || d != '{' {
		return errors.New("Expected a string or an object")
	}

	for {
		if !dec.More() {
			break
		}

		t, err := dec.Token()
		if err != nil {
			return err
		}

		switch t {
		case "name":
			err = dec.Decode(&c.Name)
		case "sort_as":
			err = dec.Decode(&c.SortAs)
		case "identifier":
			err = dec.Decode(&c.Identifier)
		case "role":
			err = dec.Decode(&c.Role)
		default:
			// Unknown field, ignore it
			dec.Token()
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Properties object use to link properties
// Use also in Rendition for fxl
type Properties struct {
	Contains     []string   `json:"contains,omitempty"`
	Layout       string     `json:"layout,omitempty"`
	MediaOverlay string     `json:"media-overlay,omitempty"`
	Orientation  string     `json:"orientation,omitempty"`
	Overflow     string     `json:"overflow,omitempty"`
	Page         string     `json:"page,omitempty"`
	Spread       string     `json:"spread,omitempty"`
	Encrypted    *Encrypted `json:"encrypted,omitempty"`
}

// Encrypted contains metadata from encryption xml
type Encrypted struct {
	Scheme         string `json:"scheme,omitempty"`
	Profile        string `json:"profile,omitempty"`
	Algorithm      string `json:"algorithm,omitempty"`
	Compression    string `json:"compression,omitempty"`
	OriginalLength int    `json:"original-length,omitempty"`
}

// Subject as based on EPUB 3.1 and WePpub
type Subject struct {
	Name   string `json:"name"`
	SortAs string `json:"sort_as,omitempty"`
	Scheme string `json:"scheme,omitempty"`
	Code   string `json:"code,omitempty"`
}

// BelongsTo is a list of collections/series that a publication belongs to
type BelongsTo struct {
	Series     []Collection `json:"series,omitempty"`
	Collection []Collection `json:"collection,omitempty"`
}

// Collection construct used for collection/serie metadata
type Collection struct {
	Name       string  `json:"name"`
	SortAs     string  `json:"sort_as,omitempty"`
	Identifier string  `json:"identifier,omitempty"`
	Position   float32 `json:"position,omitempty"`
}

// MultiLanguage store the a basic string when we only have one lang
// Store in a hash by language for multiple string representation
type MultiLanguage struct {
	SingleString string
	MultiString  map[string]string
}

// MarshalJSON overwrite json marshalling for MultiLanguage
// when we have an entry in the Multi fields we use it
// otherwise we use the single string
func (m MultiLanguage) MarshalJSON() ([]byte, error) {
	if len(m.MultiString) > 0 {
		return json.Marshal(m.MultiString)
	}
	return json.Marshal(m.SingleString)
}

func (m MultiLanguage) String() string {
	if len(m.MultiString) > 0 {
		for _, s := range m.MultiString {
			return s
		}
	}
	return m.SingleString
}

func (m *MultiLanguage) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &m.SingleString); err == nil {
		return nil
	}

	return json.Unmarshal(b, &m.MultiString)
}
