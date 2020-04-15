package rwpm

import (
	"encoding/json"
	"testing"
	"time"
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

type DateOrDatetimeStruct struct {
	Dt DateOrDatetime `json:"dt"`
}

func TestDateOrDatetime(t *testing.T) {
	var obj DateOrDatetimeStruct

	// case = the property is a date
	const date = `{"dt":"2020-04-01"}`
	if err := json.Unmarshal([]byte(date), &obj); err != nil {
		t.Fatal(err)
	} else {
		tm := time.Time(obj.Dt)
		if tm.Format(time.RFC3339) != "2020-04-01T00:00:00Z" {
			t.Errorf("Expected '2020-04-01T00:00:00Z', got %#v", tm.Format(time.RFC3339))
		}
	}
	jstring, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != `{"dt":"2020-04-01T00:00:00Z"}` {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}

	// case = the property is a date-time
	const datetime = `{"dt":"2020-04-01T01:02:03+02:00"}`
	if err := json.Unmarshal([]byte(datetime), &obj); err != nil {
		t.Fatal(err)
	} else {
		tm := time.Time(obj.Dt)
		if tm.Format(time.RFC3339) != "2020-04-01T01:02:03+02:00" {
			t.Errorf("Expected '2020-04-01T01:02:03+02:00', got %#v", tm.Format(time.RFC3339))
		}
	}
	jstring, err = json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != `{"dt":"2020-04-01T01:02:03+02:00"}` {
		t.Errorf("Expected string equality, got %#v", string(jstring))
	}
}

type DateStruct struct {
	Dt Date `json:"dt"`
}

func TestDate(t *testing.T) {

	var obj DateStruct

	const date = `{"dt":"2020-04-01"}`
	if err := json.Unmarshal([]byte(date), &obj); err != nil {
		t.Fatal(err)
	} else {
		tm := time.Time(obj.Dt)
		if tm.Format(time.RFC3339) != "2020-04-01T00:00:00Z" {
			t.Errorf("Expected '2020-04-01T00:00:00Z', got %#v", tm.Format(time.RFC3339))
		}
	}
	jstring, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if string(jstring) != `{"dt":"2020-04-01"}` {
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

type subjectStruct struct {
	Sbj Subjects `json:"dcterms:subject"`
}

func TestW3CSubjects(t *testing.T) {
	var obj subjectStruct

	// literal value
	const singleText = `{"dcterms:subject":"single"}`
	if err := json.Unmarshal([]byte(singleText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Sbj) != 1 || obj.Sbj[0].Name != "single" {
			t.Errorf("Expected one subject named single, got %#v", obj.Sbj)
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
	const arrayOfText = `{"dcterms:subject":["single","double"]}`
	if err := json.Unmarshal([]byte(arrayOfText), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Sbj) != 2 || obj.Sbj[0].Name != "single" || obj.Sbj[1].Name != "double" {
			t.Errorf("Expected 2 subjects named single and double, got %#v", obj.Sbj)
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
	const singleObject = `{"dcterms:subject":{"name":"music","scheme":"art","code":"01"}}`
	if err := json.Unmarshal([]byte(singleObject), &obj); err != nil {
		t.Fatal(err)
	} else {
		if len(obj.Sbj) != 1 || obj.Sbj[0].Name != "music" || obj.Sbj[0].Scheme != "art" || obj.Sbj[0].Code != "01" || obj.Sbj[0].SortAs != "" {
			t.Errorf("Expected 1 subject named music, got %#v", obj.Sbj)
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
