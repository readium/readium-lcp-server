package sign

import (
	"bytes"
	"testing"
)

func TestCanonSimple(t *testing.T) {
	input := make(map[string]interface{})
	input["test"] = 1
	input["abc"] = "test"
	out, err := Canon(input)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"abc":"test","test":1}`
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected %s, got %s", expected, out)
	}
}

type simpleStruct struct {
	B string
	A string
}

func TestCanonStruct(t *testing.T) {
	expected := `{"A":"1","B":"2"}`
	out, err := Canon(simpleStruct{"2", "1"})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected %s, got %s", expected, out)
	}
}

type nestedStruct struct {
	Test  string
	Inner simpleStruct
}

func TestCanonInnerStruct(t *testing.T) {
	expected := `{"Inner":{"A":"1","B":"2"},"Test":"Blah"}`
	out, err := Canon(nestedStruct{"Blah", simpleStruct{"2", "1"}})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected %s, got %s", expected, out)
	}
}
