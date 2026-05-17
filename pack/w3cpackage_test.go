// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/readium/readium-lcp-server/rwpm"
)

// TestMapW3CPublication tests the mapping of a W3C manifest to a Readium manifest
func TestMapW3CPublication(t *testing.T) {

	file, err := os.Open("./samples/w3cman1.json")
	if err != nil {
		t.Fatalf("Could not find the sample file, %s", err)
	}
	defer file.Close()

	var w3cManifest rwpm.W3CPublication
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&w3cManifest)
	if err != nil {
		t.Fatalf("Could not unmarshal sample file, %s", err)
	}

	rman := generateRWPManifest(w3cManifest)

	// metadata
	meta := rman.Metadata

	if meta.Identifier != "id1" {
		t.Fatalf("W3C Identifer badly mapped")
	}
	if meta.Title.Text() != "audiotest" {
		t.Fatalf("W3C Name badly mapped")
	}
	if meta.Publisher.Name() != "Stanford" {
		t.Fatalf("W3C Publisher badly mapped")
	}
	if meta.Author.Name() != "" {
		t.Fatalf("W3C Author badly mapped 1")
	}

	i := 0
	for _, a := range meta.Author {
		if a.Name.Text() == "Alpha" || a.Name.Text() == "Beta" || a.Name.Text() == "Gamma" {
			i++
		}
	}
	if i != 2 {
		t.Fatalf("W3C Author badly mapped, expected 2 got %d", i)
	}
	if meta.Language[0] != "fr" || meta.Language[1] != "en" {
		t.Fatalf("W3C InLanguage badly mapped")
	}
	if *meta.Published != rwpm.Date(time.Date(2020, 03, 23, 12, 50, 20, 0, time.UTC)) {
		t.Fatalf("W3C DatePublished badly mapped")
	}
	mod := time.Date(2020, 03, 23, 16, 58, 27, 372000000, time.UTC)
	if *meta.Modified != mod {
		t.Fatalf("W3C DateModified badly mapped")
	}
	if meta.Duration != 150 {
		t.Fatalf("W3C Duration badly mapped")
	}

	// Linked resources
	item0 := rman.ReadingOrder[0]
	if item0.Href != "audio/gtr-jazz.mp3" {
		t.Fatalf("W3C URL badly mapped")
	}
	if item0.Type != "audio/mpeg" {
		t.Fatalf("W3C EncodingFormat badly mapped")
	}
	if item0.Title != "Track 1" {
		t.Fatalf("W3C Name badly mapped")
	}
	if item0.Duration != 10 {
		t.Fatalf("W3C Duration badly mapped")
	}

	item1 := rman.ReadingOrder[1]
	if item1.Type != "audio/mpeg" {
		t.Fatalf("W3C EncodingFormat badly mapped if missing")
	}
	if item1.Alternate[0].Href != "audio/Latin.mp3" {
		t.Fatalf("W3C Name badly mapped in Alternate")
	}
	if item1.Alternate[0].Type != "audio/mpeg" {
		t.Fatalf("W3C EncodingFormat badly mapped in Alternate")
	}

}

// TestBuildRPFFromLPFWithoutDataDescriptors checks that BuildRPFFromLPF can
// process LPF archives whose zip entries don't carry the data-descriptor flag
// (Flags & 0x8 == 0) — the default output of Info-ZIP, 7-Zip, macOS Archive
// Utility, and Python's zipfile, so the bulk of LPF files seen in the wild.
//
// Regression: the implementation previously passed &file.FileHeader to
// zip.Writer.CreateHeader, which mutates fh.Flags |= 0x8 in place. Because the
// reader and writer shared the same FileHeader value, the subsequent
// file.Open() saw a data-descriptor flag the producer never wrote, read 12
// bytes past the file body as the descriptor, and surfaced as
// "zip: checksum error".
func TestBuildRPFFromLPFWithoutDataDescriptors(t *testing.T) {
	dir := t.TempDir()
	lpfPath := filepath.Join(dir, "test.lpf")
	rwppPath := filepath.Join(dir, "test.rwpp")

	writeLPFWithoutDataDescriptors(t, lpfPath)

	if _, err := BuildRPFFromLPF(lpfPath, rwppPath); err != nil {
		t.Fatalf("BuildRPFFromLPF: %v", err)
	}

	r, err := zip.OpenReader(rwppPath)
	if err != nil {
		t.Fatalf("open generated rwpp: %v", err)
	}
	defer r.Close()

	got := map[string]bool{}
	for _, f := range r.File {
		got[f.Name] = true
	}
	for _, want := range []string{RWPManifestName, W3CManifestName, "audio/track1.mp3"} {
		if !got[want] {
			t.Errorf("rwpp missing entry %q", want)
		}
	}
}

// writeLPFWithoutDataDescriptors writes a minimal valid W3C LPF whose entries
// all have Flags=0x0000 (no data descriptor). zip.Writer.CreateHeader always
// sets Flags |= 0x8 on non-directory entries, so we use CreateRaw and write
// store-mode entries by hand.
func writeLPFWithoutDataDescriptors(t *testing.T, path string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create lpf: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	manifest := []byte(`{` +
		`"conformsTo":"https://www.w3.org/TR/audiobooks/",` +
		`"id":"test",` +
		`"name":"Test",` +
		`"readingOrder":[{"url":"audio/track1.mp3","encodingFormat":"audio/mpeg"}]` +
		`}`)
	addStoredEntry(t, zw, W3CManifestName, manifest)
	addStoredEntry(t, zw, "audio/track1.mp3", bytes.Repeat([]byte("mp3-bytes-"), 1024))

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
}

func addStoredEntry(t *testing.T, zw *zip.Writer, name string, data []byte) {
	t.Helper()

	fh := &zip.FileHeader{
		Name:               name,
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(data),
		CompressedSize64:   uint64(len(data)),
		UncompressedSize64: uint64(len(data)),
	}
	w, err := zw.CreateRaw(fh)
	if err != nil {
		t.Fatalf("CreateRaw %q: %v", name, err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write %q: %v", name, err)
	}
}
