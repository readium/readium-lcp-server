// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/rwpm"
)

func TestOpenRPFackage(t *testing.T) {
	if _, err := OpenRPF("path-does-not-exist.lcpdf"); err == nil {
		t.Errorf("Expected to receive an error on missing file, got %s", err)
	}

	reader, err := OpenRPF("./samples/basic.lcpdf")
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

func TestWriteRPFackage(t *testing.T) {
	reader, err := OpenRPF("./samples/basic.lcpdf")
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

	writer.MarkAsEncrypted("test.pdf", 4, license.BasicProfile, "http://www.w3.org/2001/04/xmlenc#aes256-cbc")

	err = writer.Close()
	if err != nil {
		t.Fatalf("Could not close packageWriter, %s", err)
	}

	r := bytes.NewReader(b.Bytes())
	zr, err := zip.NewReader(r, int64(b.Len()))
	if err != nil {
		t.Fatalf("Could not reopen written archive, %s", err)
	}

	reader, err = NewRPFReader(zr)
	if err != nil {
		t.Fatalf("Could not read archive, %s", err)
	}

	resources := reader.Resources()
	if l := len(resources); l != 2 {
		t.Fatalf("Expected to get %d resources, got %d", 2, l)
	}

	if path := resources[1].Path(); path != "test.pdf" {
		t.Errorf("Expected resource to be named test.pdf, got %s", path)
	}

	if !resources[1].Encrypted() {
		t.Errorf("Expected resource to be encrypted")
	}

	rc, err := resources[1].Open()
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

func TestRWPM(t *testing.T) {
	var manifest rwpm.Publication

	manifest.Metadata.Identifier = "id1"
	manifest.Metadata.Title.Set("fr", "title")
	manifest.Metadata.Description = "description"
	manifest.Metadata.Published = rwpm.Date(time.Date(2020, 03, 05, 10, 00, 00, 0, time.UTC))
	manifest.Metadata.Duration = 120
	manifest.Metadata.Author.AddName("Laurent")
	manifest.Metadata.Language.Add("fr")
	manifest.Metadata.ReadingProgression = "ltr"
	manifest.Metadata.Subject.Add(rwpm.Subject{Name: "software", Scheme: "iptc", Code: "04003000"})

}
