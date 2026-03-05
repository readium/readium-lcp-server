// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/rwpm"
)

func TestOpenRPFPackage(t *testing.T) {
	if _, err := OpenRPF("path-does-not-exist.webpub"); err == nil {
		t.Errorf("Expected to receive an error on missing file, got %s", err)
	}

	reader, err := OpenRPF("./samples/basic.webpub")
	if err != nil {
		t.Fatalf("Expected to be able to open basic.webpub, got %s", err)
	}
	defer reader.Close()

	resources := reader.Resources()
	if l := len(resources); l != 1 {
		t.Errorf("Expected to get %d resources, got %d", 1, l)
	}

	if path := resources[0].Path(); path != "rwpm.pdf" {
		t.Errorf("Expected resource to be named rwpm.pdf, got %s", path)
	}
}

func TestEncryptRPF(t *testing.T) {
	// define an AES encrypter
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// create a reader on the un-encrypted readium package
	reader, err := OpenRPF("./samples/basic.webpub")
	if err != nil {
		t.Fatalf("Expected to be able to open basic.webpub, got %s", err)
	}
	defer reader.Close()

	// create the encrypted package file
	/*
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer outputFile.Close()
	*/
	var b bytes.Buffer
	// create a writer on the encrypted package
	writer, err := reader.NewWriter(&b)
	if err != nil {
		t.Fatalf("Could not build a writer, %s", err)
	}
	// encrypt resources from the input package, return the encryption key
	_, err = Process(encrypter, "", reader, writer)
	if err != nil {
		t.Fatalf("Could not encrypt the publication, %s", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Could not close the writer, %s", err)
	}
}

func TestWriteRPFackage(t *testing.T) {
	reader, err := OpenRPF("./samples/basic.webpub")
	if err != nil {
		t.Fatalf("Expected to be able to open basic.lcpdf, got %s", err)
	}

	var b bytes.Buffer
	writer, err := reader.NewWriter(&b)
	if err != nil {
		t.Fatalf("Could not build a writer, %s", err)
	}

	file, err := writer.NewFile("test.txt", "text/plain", Deflate)
	if err != nil {
		t.Fatalf("Could not create a new file, %s", err)
	}

	file.Write([]byte("test content"))

	err = file.Close()
	if err != nil {
		t.Fatalf("Could not close file, %s", err)
	}

	writer.MarkAsEncrypted("test.txt", 12, "http://www.w3.org/2001/04/xmlenc#aes256-cbc")

	err = writer.Close()
	if err != nil {
		t.Fatalf("Could not close packageWriter, %s", err)
	}

}

func TestSetMetadata(t *testing.T) {
	var manifest rwpm.Publication

	manifest.Metadata.Identifier = "id1"
	manifest.Metadata.Title.Set("fr", "title")
	manifest.Metadata.Description = "description"
	published := rwpm.Date(time.Date(2020, 03, 05, 10, 00, 00, 0, time.UTC))
	manifest.Metadata.Published = &published
	manifest.Metadata.Duration = 120
	manifest.Metadata.Author.AddName("Laurent")
	manifest.Metadata.Language.Add("fr")
	manifest.Metadata.ReadingProgression = "ltr"
	manifest.Metadata.Subject.Add(rwpm.Subject{Name: "software", Scheme: "iptc", Code: "04003000"})

	manifest.ReadingOrder = []rwpm.Link{
		{
			Href: "chapter1.html",
			Type: "application/xhtml+xml",
		},
		{
			Href: "chapter2.html",
			Type: "application/xhtml+xml",
		},
	}
	manifest.Resources = []rwpm.Link{
		{
			Rel:  []string{"stylesheet"},
			Href: "style.css",
			Type: "text/css",
		},
	}

	// simulate writing and reading back the manifest
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(&manifest)
	if err != nil {
		t.Fatalf("Could not encode manifest to JSON, %s", err)
	}

	// verify metadata

	if manifest.Metadata.Identifier != "id1" {
		t.Errorf("Expected identifier to be 'id1', got '%s'", manifest.Metadata.Identifier)
	}
	if title := manifest.Metadata.Title["fr"]; title != "title" {
		t.Errorf("Expected title to be 'title', got '%s'", title)
	}
	if manifest.Metadata.Description != "description" {
		t.Errorf("Expected description to be 'description', got '%s'", manifest.Metadata.Description)
	}
	if manifest.Metadata.Published.String() != "2020-03-05" {
		t.Errorf("Expected published to be '2020-03-05', got '%s'", manifest.Metadata.Published.String())
	}
	if manifest.Metadata.Duration != 120 {
		t.Errorf("Expected duration to be 120, got %f", manifest.Metadata.Duration)
	}
	if len(manifest.Metadata.Author) != 1 || manifest.Metadata.Author[0].Name["und"] != "Laurent" {
		t.Errorf("Expected author to be 'Laurent', got '%v'", manifest.Metadata.Author)
	}
	if len(manifest.Metadata.Language) != 1 || manifest.Metadata.Language[0] != "fr" {
		t.Errorf("Expected language to be 'fr', got '%v'", manifest.Metadata.Language)
	}
	if manifest.Metadata.ReadingProgression != "ltr" {
		t.Errorf("Expected reading progression to be 'ltr', got '%s'", manifest.Metadata.ReadingProgression)
	}
	if len(manifest.Metadata.Subject) != 1 || manifest.Metadata.Subject[0].Name != "software" {
		t.Errorf("Expected subject to be 'software', got '%v'", manifest.Metadata.Subject)
	}
	if len(manifest.ReadingOrder) != 2 || manifest.ReadingOrder[0].Href != "chapter1.html" {
		t.Errorf("Expected first reading order item to be 'chapter1.html', got '%v'", manifest.ReadingOrder)
	}
	if len(manifest.ReadingOrder) != 2 || manifest.ReadingOrder[1].Href != "chapter2.html" {
		t.Errorf("Expected second reading order item to be 'chapter2.html', got '%v'", manifest.ReadingOrder)
	}

}
