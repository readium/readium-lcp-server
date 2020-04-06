package rwpm

import (
	"encoding/json"
	"testing"
)

type multiStringStruct struct {
	Ms MultiString `json:"ms"`
}

func TestMultiString(t *testing.T) {
	var obj multiStringStruct

	const single = `{"ms":"single"}`
	if err := json.Unmarshal([]byte(single), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ms) == 0 || obj.Ms[0] != "single" {
			t.Errorf("Expected one value named single, got %#v", obj.Ms)
		}
	}
	// check  MarshalJSON
	jstring, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != single {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

}

type multiLanguageStruct struct {
	Ml MultiLanguage `json:"ml"`
}

func TestMultiLanguage(t *testing.T) {
	var obj multiLanguageStruct

	// case = the property has a literal value
	const single = `{"ml":"literal"}`
	if err := json.Unmarshal([]byte(single), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ml.Text() != "literal" || obj.Ml["und"] != "literal" {
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

	// case= the property has a single localized value
	const mapped = `{"ml":{"fr":"nom"}}`
	if err := json.Unmarshal([]byte(mapped), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ml["fr"] != "nom" {
			t.Errorf("Expected one value named 'nom', got %#v", obj.Ml["fr"])
		}
		if obj.Ml["und"] != "" {
			t.Errorf("Expected no 'und' value, got %#v", obj.Ml["und"])
		}
	}
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != mapped {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// case= the property has multiple localized values
	const multiple = `{"ml":{"en":"name","fr":"nom"}}`
	if err := json.Unmarshal([]byte(multiple), &obj); err != nil {
		t.Fatal(err)
	} else {
		if obj.Ml["en"] != "name" {
			t.Errorf("Expected one value named 'name', got %#v", obj.Ml["en"])
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

type contributorsStruct struct {
	Ctor Contributors `json:"ctor"`
}

func TestContributors(t *testing.T) {
	var obj contributorsStruct

	// check that we correctly implement UnmarshalJSON
	//var _ json.Unmarshaler = (*Contributors)(nil)

	const singleText = `{"ctor":"single"}`
	if err := json.Unmarshal([]byte(singleText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ctor) != 1 || obj.Ctor.Name() != "single" {
			t.Errorf("Expected one contributor named single, got %#v", obj.Ctor.Name())
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

	const arrayOfText = `{"ctor":["single","double"]}`
	if err := json.Unmarshal([]byte(arrayOfText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ctor) != 2 || obj.Ctor[0].Name.Text() != "single" {
			t.Errorf("Expected 2 contributors named single, got %#v", obj.Ctor)
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

	const singleObject = `{"ctor":{"name":"William Shakespeare","role":"author"}}`
	if err := json.Unmarshal([]byte(singleObject), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ctor) != 1 || obj.Ctor[0].Name.Text() != "William Shakespeare" || obj.Ctor[0].Role != "author" {
			t.Errorf("Expected 1 contributor (author) named William Shakespeare, got %#v", obj.Ctor[0].Name.Text())
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

	const multilingualObject = `{"ctor":{"name":{"en":"Mikhail Bulgakov","fr":"Mikhaïl Boulgakov","ru":"Михаил Афанасьевич Булгаков"}}}`
	if err := json.Unmarshal([]byte(multilingualObject), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ctor) != 1 || obj.Ctor[0].Name["fr"] != "Mikhaïl Boulgakov" {
			t.Errorf("Expected 1 contributor (author) with a French named of Mikhaïl Boulgakov, got %#v", obj.Ctor)
		}
	}
	// check MarshalJSON
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	// string equality is respected because localized names propertly ordered in the sample
	if string(jstring) != multilingualObject {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	const arrayOfObjects = `{"ctor":[{"name":"William Shakespeare","role":"author"},{"name":"Virginia Wolfe","identifier":"12345"},"Jean-Jacques Rousseau"]}`
	if err := json.Unmarshal([]byte(arrayOfObjects), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Ctor) != 3 || obj.Ctor[0].Name.Text() != "William Shakespeare" || obj.Ctor[0].Role != "author" {
			t.Errorf("Expected 3 contributors, got %#v", obj.Ctor)
		}
	}
	// check MarshalJSON
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != arrayOfObjects {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

}
