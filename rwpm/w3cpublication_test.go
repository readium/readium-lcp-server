// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package rwpm

import (
	"encoding/json"
	"testing"
)

type W3CmultiLanguageStruct struct {
	Ml W3CMultiLanguage `json:"ml"`
}

func TestW3CMultiLanguage(t *testing.T) {
	var obj W3CmultiLanguageStruct

	// case = the property has a literal value
	const single = `{"ml":"literal"}`
	if err := json.Unmarshal([]byte(single), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ml.Text() != "literal" {
			t.Errorf("Expected one value named 'literal', got %#v", obj.Ml)
		}
	}
	jstring, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != single {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// case = the property has multiple localized values
	const multiple = `{"ml":[{"language":"fr","value":"nom","direction":"ltr"},{"language":"en","value":"name","direction":"ltr"}]}`
	if err := json.Unmarshal([]byte(multiple), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ml[0].Value != "nom" {
			t.Errorf("Expected one value named 'nom', got %#v", obj.Ml[0].Value)
		}
	}
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != multiple {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}
}
