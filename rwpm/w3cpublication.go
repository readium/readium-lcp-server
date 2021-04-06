// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package rwpm

import (
	"encoding/json"
)

// W3CPublication = W3C manifest
type W3CPublication struct {
	ConformsTo         string           `json:"conformsTo,omitempty"`
	ID                 string           `json:"id,omitempty"`
	URL                string           `json:"url,omitempty"`
	Name               W3CMultiLanguage `json:"name"`
	Publisher          W3CContributors  `json:"publisher,omitempty"`
	Artist             W3CContributors  `json:"artist,omitempty"`
	Author             W3CContributors  `json:"author,omitempty"`
	Colorist           W3CContributors  `json:"colorist,omitempty"`
	Contributor        W3CContributors  `json:"contributor,omitempty"`
	Creator            W3CContributors  `json:"creator,omitempty"`
	Editor             W3CContributors  `json:"editor,omitempty"`
	Illustrator        W3CContributors  `json:"illustrator,omitempty"`
	Inker              W3CContributors  `json:"inker,omitempty"`
	Letterer           W3CContributors  `json:"letterer,omitempty"`
	Penciler           W3CContributors  `json:"penciler,omitempty"`
	ReadBy             W3CContributors  `json:"readBy,omitempty"`
	Translator         W3CContributors  `json:"translator,omitempty"`
	InLanguage         MultiString      `json:"inLanguage,omitempty"`
	DatePublished      *DateOrDatetime  `json:"datePublished,omitempty"`
	DateModified       *DateOrDatetime  `json:"dateModified,omitempty"`
	ReadingProgression string           `json:"readingProgression,omitempty"`
	Duration           string           `json:"duration,omitempty"`
	Description        string           `json:"dcterms:description,omitempty"`
	Subject            Subjects         `json:"dcterms:subject,omitempty"`
	Links              W3CLinks         `json:"links,omitempty"`
	ReadingOrder       W3CLinks         `json:"readingOrder,omitempty"`
	Resources          W3CLinks         `json:"resources,omitempty"`
}

// W3CLink object
type W3CLink struct {
	URL            string           `json:"url"`
	EncodingFormat string           `json:"encodingFormat,omitempty"`
	Name           W3CMultiLanguage `json:"name,omitempty"`
	Description    W3CMultiLanguage `json:"description,omitempty"`
	Rel            MultiString      `json:"rel,omitempty"`
	Integrity      string           `json:"integrity,omitempty"`
	Duration       string           `json:"duration,omitempty"`
	Alternate      W3CLinks         `json:"alternate,omitempty"`
}

// W3CContributors is an array of contributors
type W3CContributors []W3CContributor

// UnmarshalJSON unmarshals contributors
func (c *W3CContributors) UnmarshalJSON(b []byte) error {

	var ctors []W3CContributor
	ctors = make([]W3CContributor, 1)
	var ctor W3CContributor

	// literal value
	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		ctors[0].Name = make([]W3CLocalized, 1)
		ctors[0].Name[0] = W3CLocalized{"und", literal, ""}

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

// MarshalJSON marshals W3CContributors
func (c W3CContributors) MarshalJSON() ([]byte, error) {

	// literal value
	if len(c) == 1 && c[0].Name[0].Language == "und" && c[0].Name[0].Value != "" && c[0].ID == "" &&
		c[0].URL == "" && c[0].Identifier == nil {
		return json.Marshal(c[0].Name.Text())
	}

	// object value
	if len(c) == 1 {
		ctor := c[0]
		return json.Marshal(ctor)
	}

	// array value
	var ctors []W3CContributor
	ctors = c
	return json.Marshal(ctors)
}

// W3CContributor construct used internally for all contributors
type W3CContributor struct {
	Type       MultiString      `json:"type,omitempty"`
	Name       W3CMultiLanguage `json:"name,omitempty"`
	ID         string           `json:"id,omitempty"`
	URL        string           `json:"url,omitempty"`
	Identifier MultiString      `json:"identifier,omitempty"`
}

// UnmarshalJSON unmarshals W3CContributor
func (c *W3CContributor) UnmarshalJSON(b []byte) error {

	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		c.Name = make([]W3CLocalized, 1)
		c.Type = nil
		c.Name[0] = W3CLocalized{"und", literal, ""}
		c.ID = ""
		c.URL = ""
		c.Identifier = nil
		return nil
	}
	type Alias W3CContributor
	var ctorAlias Alias
	err = json.Unmarshal(b, &ctorAlias)
	if err != nil {
		return err
	}
	*c = W3CContributor(ctorAlias)
	return nil
}

// MarshalJSON marshals W3CContributor
func (c W3CContributor) MarshalJSON() ([]byte, error) {

	// literal value
	if c.Name[0].Language == "und" && c.Name[0].Value != "" && c.ID == "" && c.URL == "" && c.Identifier == nil {
		return json.Marshal(c.Name[0].Value)
	}
	type Alias W3CContributor
	ctorAlias := Alias{c.Type, c.Name, c.ID, c.URL, c.Identifier}
	return json.Marshal(ctorAlias)
}

// W3CMultiLanguage struct
type W3CMultiLanguage []W3CLocalized

// W3CLocalized represents a single value of a W3CMultiLanguage property
type W3CLocalized struct {
	Language  string `json:"language"`
	Value     string `json:"value"`
	Direction string `json:"direction,omitempty"`
}

// UnmarshalJSON unmarshalls W3CMultilanguge
func (m *W3CMultiLanguage) UnmarshalJSON(b []byte) error {

	var locs []W3CLocalized
	locs = make([]W3CLocalized, 1)
	var loc W3CLocalized
	var literal string
	var err error

	// literal value
	if err = json.Unmarshal(b, &literal); err == nil {
		locs[0] = W3CLocalized{"und", literal, ""}

		// object value
	} else if err = json.Unmarshal(b, &loc); err == nil {
		locs[0] = loc

		// array value
	} else {
		err = json.Unmarshal(b, &locs)
	}
	if err == nil {
		*m = locs
		return nil
	}
	return err
}

// MarshalJSON marshalls W3CMultiLanguage
func (m W3CMultiLanguage) MarshalJSON() ([]byte, error) {

	// literal value
	if len(m) == 1 && m[0].Language == "und" &&
		m[0].Value != "" && m[0].Direction == "" {
		return json.Marshal(m[0].Value)
	}

	// object value
	if len(m) == 1 {
		loc := m[0]
		return json.Marshal(loc)
	}

	// array value
	locs := []W3CLocalized(m)
	return json.Marshal(locs)
}

// Text returns the "und" language value or the first value found in the map
func (m W3CMultiLanguage) Text() string {

	for _, ml := range m {
		if ml.Language == "und" {
			return ml.Value
		}
	}
	return m[0].Value
}

// UnmarshalJSON unmarshalls W3CLocalized
func (l *W3CLocalized) UnmarshalJSON(b []byte) error {

	var literal string
	var err error

	// literal value
	if err = json.Unmarshal(b, &literal); err == nil {
		*l = W3CLocalized{"und", literal, ""}
		return nil

	}
	// object value
	type Alias W3CLocalized
	var locAlias Alias
	err = json.Unmarshal(b, &locAlias)
	if err != nil {
		return err
	}
	*l = W3CLocalized(locAlias)
	return nil

}

// MarshalJSON marshalls W3CLocalized
func (l W3CLocalized) MarshalJSON() ([]byte, error) {

	// literal value
	if l.Language == "und" && l.Value != "" && l.Direction == "" {
		return json.Marshal(l.Value)
	}
	// object value
	type Alias W3CLocalized
	locAlias := Alias{l.Language, l.Value, l.Direction}
	return json.Marshal(locAlias)
}

// W3CLinks struct
type W3CLinks []W3CLink

// UnmarshalJSON unmarshalls W3CLinks
func (l *W3CLinks) UnmarshalJSON(b []byte) error {

	var lnks []W3CLink
	lnks = make([]W3CLink, 1)
	var lnk W3CLink

	// literal value
	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		lnks[0].URL = literal

		// object value
	} else if err = json.Unmarshal(b, &lnk); err == nil {
		lnks[0] = lnk

		// array value
	} else {
		err = json.Unmarshal(b, &lnks)
	}
	if err == nil {
		*l = lnks
		return nil
	}
	return err
}

// MarshalJSON marshals W3CLinks
func (l W3CLinks) MarshalJSON() ([]byte, error) {

	// literal value
	if len(l) == 1 && l[0].URL != "" &&
		l[0].EncodingFormat == "" && l[0].Name == nil && l[0].Description == nil &&
		l[0].Rel == nil && l[0].Integrity == "" && l[0].Duration == "" && l[0].Alternate == nil {
		return json.Marshal(l[0].URL)
	}

	// object value
	if len(l) == 1 {
		lnk := l[0]
		return json.Marshal(lnk)
	}

	// array value
	var lnks []W3CLink
	lnks = l
	return json.Marshal(lnks)
}

// UnmarshalJSON unmarshals W3CLink
func (l *W3CLink) UnmarshalJSON(b []byte) error {

	var literal string
	var err error

	// literal value
	if err = json.Unmarshal(b, &literal); err == nil {
		*l = W3CLink{URL: literal}
		return nil
	}
	// object vcalue
	type Alias W3CLink
	var lnkAlias Alias
	err = json.Unmarshal(b, &lnkAlias)
	if err != nil {
		return err
	}
	*l = W3CLink(lnkAlias)
	return nil
}

// MarshalJSON marshals W3CLink
func (l W3CLink) MarshalJSON() ([]byte, error) {

	// literal value
	if l.URL != "" && l.EncodingFormat == "" && l.Name == nil && l.Description == nil &&
		l.Rel == nil && l.Integrity == "" && l.Duration == "" && l.Alternate == nil {
		return json.Marshal(l.URL)
	}
	type Alias W3CLink
	lnkAlias := Alias{l.URL, l.EncodingFormat, l.Name, l.Description, l.Rel, l.Integrity, l.Duration, l.Alternate}
	return json.Marshal(lnkAlias)
}
