// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package rwpm

import (
	"encoding/json"
	"strings"
	"time"
)

// Metadata for the default context in WebPub
type Metadata struct {
	Type               string        `json:"@type,omitempty"`
	Identifier         string        `json:"identifier,omitempty"`
	Title              MultiLanguage `json:"title"`
	Subtitle           MultiLanguage `json:"subtitle,omitempty"`
	SortAs             string        `json:"sortAs,omitempty"`
	Description        string        `json:"description,omitempty"`
	Language           MultiString   `json:"language,omitempty"`
	ReadingProgression string        `json:"readingProgression,omitempty"`
	Modified           time.Time     `json:"modified,omitempty"`
	Published          Date          `json:"published,omitempty"`
	// contributors
	Publisher   Contributors `json:"publisher,omitempty"`
	Artist      Contributors `json:"artist,omitempty"`
	Author      Contributors `json:"author,omitempty"`
	Colorist    Contributors `json:"colorist,omitempty"`
	Contributor Contributors `json:"contributor,omitempty"`
	Editor      Contributors `json:"editor,omitempty"`
	Illustrator Contributors `json:"illustrator,omitempty"`
	Imprint     Contributors `json:"imprint,omitempty"`
	Inker       Contributors `json:"inker,omitempty"`
	Letterer    Contributors `json:"letterer,omitempty"`
	Narrator    Contributors `json:"narrator,omitempty"`
	Penciler    Contributors `json:"penciler,omitempty"`
	Translator  Contributors `json:"translator,omitempty"`
	// other descriptive metadata
	Subject       Subjects `json:"subject,omitempty"`
	Duration      int      `json:"duration,omitempty"`
	NumberOfPages int      `json:"numberOfPages,omitempty"`
	Abridged      bool     `json:"abridged,omitempty"`
	// collections & series
	BelongsTo *BelongsTo `json:"belongsTo,omitempty"`

	OtherMetadata []Meta `json:"-"` //Extension point for other metadata
}

// DateOrDatetime struct
type DateOrDatetime time.Time

// UnmarshalJSON unmarshalls DateOrDatetime
func (d *DateOrDatetime) UnmarshalJSON(b []byte) error {

	s := strings.Trim(string(b), "\"")
	// process a date
	if len(s) == 11 && strings.Index(s, "Z") == 10 { // a date may end with a 'Z'
		s = strings.TrimRight(s, "Z")
	}
	if len(s) == 10 {
		s = s + "T00:00:00Z"
	}

	// process a date-time, RFC 3999 compliant
	date, err := time.Parse(time.RFC3339, s)
	*d = DateOrDatetime(date)
	return err
}

// MarshalJSON marshalls DateOrDatetime
func (d DateOrDatetime) MarshalJSON() ([]byte, error) {

	return json.Marshal(time.Time(d))
}

// Date struct
type Date time.Time

// UnmarshalJSON unmarshalls Date
func (d *Date) UnmarshalJSON(b []byte) error {

	// trim the quotes around the value
	s := string(b[1 : len(b)-1])
	// process a date
	if len(s) == 11 && strings.Index(s, "Z") == 10 { // a date may end with a 'Z'
		s = strings.TrimRight(s, "Z")
	}
	if len(s) == 10 {
		s = s + "T00:00:00Z"
	}

	// process a date-time, RFC 3999 compliant
	date, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*d = Date(date)
	return nil
}

// MarshalJSON marshalls Date
func (d Date) MarshalJSON() ([]byte, error) {

	date := time.Time(d)
	return []byte(date.Format("\"2006-01-02\"")), nil
}

// Meta is a generic structure for other metadata
type Meta struct {
	Property string
	Value    interface{}
	Children []Meta
}

// Properties object used to link properties
// Used also in Rendition for fxl
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

// Subjects is an array of subjects
type Subjects []Subject

// Subject of a publication
type Subject struct {
	Name   string `json:"name"`
	SortAs string `json:"sort_as,omitempty"`
	Scheme string `json:"scheme,omitempty"`
	Code   string `json:"code,omitempty"`
}

// Add adds a value to a subject array
func (s *Subjects) Add(value Subject) {

	*s = append(*s, value)
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

// Contributors is an array of contributors
type Contributors []Contributor

// UnmarshalJSON unmarshals contributors
func (c *Contributors) UnmarshalJSON(b []byte) error {

	var ctors []Contributor
	ctors = make([]Contributor, 1)
	var ctor Contributor

	// literal value
	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		ctors[0].Name.SetDefault(literal)

		// object value
	} else if err = json.Unmarshal(b, &ctor); err == nil {
		ctors[0] = ctor

		// array value
	} else {
		err = json.Unmarshal(b, &ctors)
	}
	if err == nil {
		*c = ctors
		return nil
	}
	return err
}

// MarshalJSON marshals Contributors
func (c Contributors) MarshalJSON() ([]byte, error) {

	// literal value
	if len(c) == 1 && c[0].Name.Text() != "" &&
		c[0].Identifier == "" && c[0].SortAs == "" && c[0].Role == "" {
		return json.Marshal(c[0].Name.Text())
	}

	// object value
	if len(c) == 1 {
		ctor := c[0]
		return json.Marshal(ctor)
	}

	// array value
	var ctors []Contributor
	ctors = c
	return json.Marshal(ctors)
}

// AddName adds a Contributor to Contributors
func (c *Contributors) AddName(name string) {

	var ctor Contributor
	ctor.Name.SetDefault(name)
	c.Add(ctor)
}

// Add adds a Contributor to Contributors
func (c *Contributors) Add(ctor Contributor) {

	*c = append(*c, ctor)
}

// Name gets the name of a contributor
func (c Contributors) Name() string {

	if len(c) == 1 && c[0].Name.Text() != "" {
		return c[0].Name.Text()
	}
	return ""
}

// Contributor construct used internally for all contributors
type Contributor struct {
	Name       MultiLanguage `json:"name,omitempty"`
	SortAs     string        `json:"sortAs,omitempty"`
	Identifier string        `json:"identifier,omitempty"`
	Role       string        `json:"role,omitempty"`
}

// UnmarshalJSON unmarshals Contributor
func (c *Contributor) UnmarshalJSON(b []byte) error {

	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		c.Name = make(map[string]string)
		c.Name["und"] = literal
		c.SortAs = ""
		c.Identifier = ""
		c.Role = ""
		return nil
	}
	type Alias Contributor
	var ctorAlias Alias
	err = json.Unmarshal(b, &ctorAlias)
	if err != nil {
		return err
	}
	*c = Contributor(ctorAlias)
	return nil
}

// MarshalJSON marshals Contributor
func (c Contributor) MarshalJSON() ([]byte, error) {

	// literal value
	if c.Name["und"] != "" && c.Identifier == "" && c.Role == "" && c.SortAs == "" {
		return json.Marshal(c.Name["und"])
	}
	type Alias Contributor
	ctorAlias := Alias{c.Name, c.SortAs, c.Identifier, c.Role}
	return json.Marshal(ctorAlias)
}

// MultiLanguage stores one or more values indexed by language.
type MultiLanguage map[string]string

// UnmarshalJSON unmarshalls Multilanguge
// The "und" (undefined)language corresponds to a literal value
func (m *MultiLanguage) UnmarshalJSON(b []byte) error {

	var mmap map[string]string
	mmap = make(map[string]string)

	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		mmap["und"] = literal
	} else {
		err = json.Unmarshal(b, &mmap)
	}
	if err != nil {
		return err
	}
	*m = mmap
	return nil
}

// MarshalJSON marshalls MultiLanguage
func (m MultiLanguage) MarshalJSON() ([]byte, error) {

	if len(m) > 1 || m["und"] == "" {
		var mmap map[string]string
		mmap = make(map[string]string)

		for key, value := range m {
			mmap[key] = value
		}
		return json.Marshal(mmap)
	}
	return json.Marshal(m["und"])
}

// Text returns the "und" language value or the single value found in the map
func (m MultiLanguage) Text() string {

	if m["und"] != "" {
		return m["und"]
	} else if len(m) == 1 {
		for _, v := range m {
			return v
		}
	}
	return ""
}

// SetDefault inits the "und" localized value
func (m *MultiLanguage) SetDefault(literal string) {

	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)["und"] = literal
}

// Set inits a localized value
func (m *MultiLanguage) Set(language string, value string) {

	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)[language] = value
}

// MultiString stores one or more strings
// Used for properties which take a string || an array of strings
type MultiString []string

// UnmarshalJSON unmarshals MultiString
func (m *MultiString) UnmarshalJSON(b []byte) error {

	var mstring []string
	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		mstring = append(mstring, literal)
	} else {
		err = json.Unmarshal(b, &mstring)
	}
	if err != nil {
		return err
	}
	*m = mstring
	return nil
}

// MarshalJSON marshalls MultiString
func (m MultiString) MarshalJSON() ([]byte, error) {

	if len(m) == 1 {
		literal := m[0]
		return json.Marshal(literal)
	}
	var mstring []string
	for _, v := range m {
		mstring = append(mstring, v)
	}
	return json.Marshal(mstring)
}

// Add adds a value to a multistring array
func (m *MultiString) Add(value string) {

	*m = append(*m, value)
}
