package pack

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"testing"
)

func TestOpenRWPPackage(t *testing.T) {
	if _, err := OpenPackagedRWP("path-does-not-exist.lcpdf"); err == nil {
		t.Errorf("Expected to receive an error on missing file, got %s", err)
	}

	reader, err := OpenPackagedRWP("./samples/basic.lcpdf")
	if err != nil {
		t.Fatalf("Expected to be able to open basic.lcpdf, got %s", err)
	}

	resources := reader.Resources()
	if l := len(resources); l != 1 {
		t.Errorf("Expected to get %d resources, got %d", 1, l)
	}

	if path := resources[0].Path(); path != "rwpm.pdf" {
		t.Errorf("Expected resource to be named rwpm.pdf, got %s", path)
	}
}

func TestWriteRWPPackage(t *testing.T) {
	reader, err := OpenPackagedRWP("./samples/basic.lcpdf")
	if err != nil {
		t.Fatalf("Expected to be able to open basic.lcpdf, got %s", err)
	}

	var b bytes.Buffer
	writer, err := reader.NewWriter(&b)
	if err != nil {
		t.Fatalf("Could not build a writer, %s", err)
	}

	file, err := writer.NewFile("test.pdf", "application/pdf", Deflate)
	if err != nil {
		t.Fatalf("Could not create a new file, %s", err)
	}

	file.Write([]byte("test"))

	err = file.Close()
	if err != nil {
		t.Fatalf("Could not close file, %s", err)
	}

	writer.MarkAsEncrypted("test.pdf", 4, "http://readium.org/lcp/basic-profile", "http://www.w3.org/2001/04/xmlenc#aes256-cbc")

	err = writer.Close()
	if err != nil {
		t.Fatalf("Could not close packageWriter, %s", err)
	}

	r := bytes.NewReader(b.Bytes())
	zr, err := zip.NewReader(r, int64(b.Len()))
	if err != nil {
		t.Fatalf("Could not reopen written archive, %s", err)
	}

	reader, err = NewPackagedRWPReader(zr)
	if err != nil {
		t.Fatalf("Could not read archive, %s", err)
	}

	resources := reader.Resources()
	if l := len(resources); l != 1 {
		t.Fatalf("Expected to get %d resources, got %d", 1, l)
	}

	if path := resources[0].Path(); path != "test.pdf" {
		t.Errorf("Expected resource to be named test.pdf, got %s", path)
	}

	if !resources[0].Encrypted() {
		t.Errorf("Expected resource to be encrypted")
	}

	rc, err := resources[0].Open()
	if err != nil {
		t.Fatalf("Could not open file: %s", err)
	}

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		t.Fatalf("Could not read data from file: %s", err)
	}

	if !bytes.Equal(data, []byte("test")) {
		t.Errorf("Bytes were not equal")
	}

}
