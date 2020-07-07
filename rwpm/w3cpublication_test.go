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

	// case = the property has a structured localized value
	const objectLoc = `{"ml":{"language":"fr","value":"nom","direction":"ltr"}}`
	if err := json.Unmarshal([]byte(objectLoc), &obj); err != nil {
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
	if string(jstring) != objectLoc {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// case = the property has multiple localized values
	const multiple = `{"ml":[{"language":"fr","value":"nom","direction":"ltr"},{"language":"en","value":"name","direction":"ltr"},"other name"]}`
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

type linkStruct struct {
	Lnk W3CLinks `json:"links"`
}

func TestW3CLinks(t *testing.T) {
	var obj linkStruct

	// literal value
	const singleText = `{"links":"single"}`
	if err := json.Unmarshal([]byte(singleText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Lnk) != 1 || obj.Lnk[0].URL != "single" {
			t.Errorf("Expected one link named single, got %#v", obj.Lnk)
		}
	}
	// check  MarshalJSON
	jstring, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != singleText {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// array of literal values
	const arrayOfText = `{"links":["single","double"]}`
	if err := json.Unmarshal([]byte(arrayOfText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Lnk) != 2 || obj.Lnk[0].URL != "single" || obj.Lnk[1].URL != "double" {
			t.Errorf("Expected 2 links named single and double, got %#v", obj.Lnk)
		}
	}
	// check MarshalJSON
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != arrayOfText {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// struct value
	const singleObject = `{"links":{"url":"linkurl","name":"linkname","rel":"relation"}}`
	if err := json.Unmarshal([]byte(singleObject), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Lnk) != 1 || obj.Lnk[0].URL != "linkurl" || obj.Lnk[0].Rel.Text() != "relation" || obj.Lnk[0].Name.Text() != "linkname" {
			t.Errorf("Expected 1 link named linkurl, got %#v", obj.Lnk)
		}
	}
	// check MarshalJSON
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != singleObject {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

}
