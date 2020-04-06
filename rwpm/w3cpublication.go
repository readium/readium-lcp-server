// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package rwpm

import "encoding/json"

// W3CPublication = W3C manifest
type W3CPublication struct {
	ConformsTo  string
	ID          string
	URL         string
	Name        W3CMultiLanguage
	Publisher   Contributors
	Artist      Contributors
	Author      Contributors
	Colorist    Contributors
	Contributor Contributors
	Creator     Contributors
	Editor      Contributors
	Illustrator Contributors
	Inker       Contributors
	Letterer    Contributors
	Penciler    Contributors
	ReadBy      Contributors
	Translator  Contributors
	InLanguage  MultiString
	// FIXME: mapping to date or date-time
	DatePublished      string
	DateModified       string
	ReadingProgression string
	Duration           string
	Description        string    `json:"dcterms:description,omitempty"`
	Subject            []Subject `json:"dcterms:subject,omitempty"`
	Links              []W3CLink
	ReadingOrder       []W3CLink
	Resources          []W3CLink
}

// W3CLink object
type W3CLink struct {
	URL            string
	EncodingFormat string
	Name           W3CMultiLanguage
	Description    W3CMultiLanguage
	Rel            MultiString
	Integrity      string
	Duration       string
	Alternate      []W3CLink
}

// W3CMultiLanguage object
type W3CMultiLanguage []W3CLocalized

// W3CLocalized represents a single value of a W3CMultiLanguage property
type W3CLocalized struct {
	Language  string `json:"language"`
	Value     string `json:"value"`
	Direction string `json:"direction,omitempty"`
}

// UnmarshalJSON unmarshalls W3CMultilanguge
func (m *W3CMultiLanguage) UnmarshalJSON(b []byte) error {

	var mloc []W3CLocalized
	var literal string
	var err error
	if err = json.Unmarshal(b, &literal); err == nil {
		mloc = append(mloc, W3CLocalized{"und", literal, ""})
	} else {
		err = json.Unmarshal(b, &mloc)
	}
	if err != nil {
		return err
	}
	*m = mloc
	return nil
}

// MarshalJSON marshalls W3CMultiLanguage
func (m W3CMultiLanguage) MarshalJSON() ([]byte, error) {

	if len(m) > 1 || m[0].Language != "und" {
		var mloc []W3CLocalized
		mloc = []W3CLocalized(m)
		return json.Marshal(mloc)
	}
	return json.Marshal(m[0].Value)
}

// Text returns the "und" language value or the single value found in thge map
func (m W3CMultiLanguage) Text() string {

	for _, ml := range m {
		if ml.Language == "und" {
			return ml.Value
		}
	}
	if len(m) == 1 {
		return m.Text()
	}
	return ""
}
