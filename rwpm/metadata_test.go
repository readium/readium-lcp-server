package rwpm

import (
	"encoding/json"
	"testing"
)

type contributorsStruct struct {
	Ctor Contributors `json:"ctor"`
}

func TestContributors(t *testing.T) {
	var obj contributorsStruct

	// check that we correctly implement UnmarshalJSON
	//var _ json.Unmarshaler = (*Contributors)(nil)

	const singleText = `{ "ctor": "single" }`
	if err := json.Unmarshal([]byte(singleText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ctor == nil || len(obj.Ctor) != 1 || obj.Ctor[0].Name.SingleString != "single" {
			t.Errorf("Expected one contributor named single, got %#v", obj.Ctor)
		}
	}

	const arrayOfText = `{ "ctor": ["single", "single"] }`
	if err := json.Unmarshal([]byte(arrayOfText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ctor == nil || len(obj.Ctor) != 2 || obj.Ctor[0].Name.SingleString != "single" {
			t.Errorf("Expected 2 contributors named single, got %#v", obj.Ctor)
		}
	}

	const singleObject = `{ "ctor": { "name": "William Shakespeare", "role": "author" } }`
	if err := json.Unmarshal([]byte(singleObject), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ctor == nil || len(obj.Ctor) != 1 || obj.Ctor[0].Name.SingleString != "William Shakespeare" || obj.Ctor[0].Role != "author" {
			t.Errorf("Expected 1 contributor (author) named William Shakespeare, got %#v", obj.Ctor)
		}
	}

	const arrayOfObjects = `{ "ctor": [{ "name": "William Shakespeare", "role": "author" }, { "name": "Virginia Wolfe" }] }`
	if err := json.Unmarshal([]byte(arrayOfObjects), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ctor == nil || len(obj.Ctor) != 2 || obj.Ctor[0].Name.SingleString != "William Shakespeare" || obj.Ctor[0].Role != "author" {
			t.Errorf("Expected 2 contributors, got %#v", obj.Ctor)
		}
	}

	const arrayOfMixedObjects = `{ "ctor": [{ "name": "William Shakespeare", "role": "author" }, { "name": "Virginia Wolfe" }, "Jean-Jacques Rousseau"] }`
	if err := json.Unmarshal([]byte(arrayOfMixedObjects), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ctor == nil || len(obj.Ctor) != 3 || obj.Ctor[0].Name.SingleString != "William Shakespeare" || obj.Ctor[0].Role != "author" {
			t.Errorf("Expected 3 contributors, got %#v", obj.Ctor)
		}
	}
}
