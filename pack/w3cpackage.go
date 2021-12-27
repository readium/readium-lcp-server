// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/rwpm"
	"github.com/rickb777/date/period"
)

// W3CManifestName is the name of the W3C manifest in an LPF package
const W3CManifestName = "publication.json"

// W3CEntryPageName is the name of the W3C entry page in an LPF package
const W3CEntryPageName = "index.html"

// RWPManifestName is the name of the Readium Manifest in a package
const RWPManifestName = "manifest.json"

// displayW3CMan displays a serialized W3C Manifest (debug purposes only)
func displayW3CMan(w3cman rwpm.W3CPublication) error {

	json, err := json.MarshalIndent(w3cman, "", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(json))
	return nil
}

// mapXlanglProperty maps a multilingual property (e.g. name)
// from a W3C manifest to a Readium manifest.
// Note: 'direction' cannot be mapped.
func mapXlanglProperty(w3clp rwpm.W3CMultiLanguage) (ml rwpm.MultiLanguage) {

	ml = make(map[string]string)
	for _, p := range w3clp {
		ml[p.Language] = p.Value
	}
	return
}

// mapContributor maps a Contributors property (e.g. author)
// from a W3C manifest to a Readium manifest.
// Note: ID is mapped to Identifier
func mapContributor(w3cctors rwpm.W3CContributors) (ctors rwpm.Contributors) {

	ctors = make(rwpm.Contributors, len(w3cctors))
	for i, c := range w3cctors {
		ctors[i].Name = mapXlanglProperty(c.Name)
		ctors[i].Identifier = c.ID
	}
	return
}

// getMediaType infers a media type from a file extension
// the media type is mandatory in the Readium manifest;
// if a media type is missing in the input link,
// try to infer it from the file extension.
// Note: a test on the magic number of the input file could be added.
func getMediaType(ext string) (mt string) {

	switch ext {
	case ".mp3":
		mt = "audio/mpeg"
	case ".aac":
		mt = "audio/aac"
	case ".opus":
		mt = "audio/ogg"
	case ".wav":
		mt = "audio/wav"
	case ".jpeg":
		mt = "image/jpeg"
	case ".jpg":
		mt = "image/jpeg"
	case ".png":
		mt = "image/png"
	case ".gif":
		mt = "image/gif"
	case ".webp":
		mt = "image/webp"
	case ".json":
		mt = "application/json"
	case ".html":
		mt = "text/html"
	case ".css":
		mt = "text/css"
	case ".js":
		mt = "application/javascript"
	case ".epub":
		mt = "application/epub+zip"
	case ".pdf":
		mt = "application/pdf"
	}
	return
}

// mapLinks copies a collection of links (reading order, resources ...)
// from a W3C manifest to a Readium manifest
func mapLinks(w3clinks []rwpm.W3CLink) (rwpmLinks []rwpm.Link) {

	for _, w3cl := range w3clinks {
		var rwpml rwpm.Link
		rwpml.Href = w3cl.URL
		if w3cl.EncodingFormat != "" {
			rwpml.Type = w3cl.EncodingFormat
		} else {
			rwpml.Type = getMediaType(filepath.Ext(w3cl.URL))
		}
		rwpml.Rel = w3cl.Rel
		// a multilingual name is lost during mapping
		if w3cl.Name != nil {
			rwpml.Title = w3cl.Name.Text()
		}
		rwpml.Duration, _ = isoDurationToSc(w3cl.Duration)

		rwpml.Alternate = mapLinks(w3cl.Alternate)

		rwpmLinks = append(rwpmLinks, rwpml)
	}
	return
}

// generateRWPManifest generates a json Readium manifest (as []byte) out of a W3C Manifest
func generateRWPManifest(w3cman rwpm.W3CPublication) (manifest rwpm.Publication) {

	// debug
	//displayW3CMan(w3cman)

	manifest.Context = []string{"https://readium.org/webpub-manifest/context.jsonld"}

	if w3cman.ConformsTo == "https://www.w3.org/TR/audiobooks/" {
		manifest.Metadata.Type = "https://schema.org/Audiobook"
		manifest.Metadata.ConformsTo = "https://readium.org/webpub-manifest/profiles/audiobook"
	} else {
		manifest.Metadata.Type = "https://schema.org/CreativeWork"
	}

	var identifier string
	if w3cman.ID != "" {
		identifier = w3cman.ID
	} else if w3cman.URL != "" {
		identifier = w3cman.URL
	} else {
		identifier, _ = newUUID()
	}
	manifest.Metadata.Identifier = identifier
	manifest.Metadata.Title = mapXlanglProperty(w3cman.Name)
	manifest.Metadata.Description = w3cman.Description
	manifest.Metadata.Subject = w3cman.Subject
	manifest.Metadata.Language = w3cman.InLanguage
	// W3C manifest: published and modified are date-or-datetime,
	// Readium manifest: published is a date; modified is a datetime
	// The use of pointer helps dealing with nil values
	if w3cman.DatePublished != nil {
		published := rwpm.Date(time.Time(*w3cman.DatePublished))
		manifest.Metadata.Published = &published
	}
	if w3cman.DateModified != nil {
		modified := time.Time(*w3cman.DateModified)
		manifest.Metadata.Modified = &modified
	}
	manifest.Metadata.Duration, _ = isoDurationToSc(w3cman.Duration)
	manifest.Metadata.ReadingProgression = w3cman.ReadingProgression

	manifest.Metadata.Publisher = mapContributor(w3cman.Publisher)
	manifest.Metadata.Artist = mapContributor(w3cman.Artist)
	manifest.Metadata.Author = mapContributor(w3cman.Author)
	manifest.Metadata.Colorist = mapContributor(w3cman.Colorist)
	manifest.Metadata.Contributor = mapContributor(w3cman.Contributor)
	manifest.Metadata.Editor = mapContributor(w3cman.Editor)
	manifest.Metadata.Illustrator = mapContributor(w3cman.Illustrator)
	manifest.Metadata.Inker = mapContributor(w3cman.Inker)
	manifest.Metadata.Letterer = mapContributor(w3cman.Letterer)
	manifest.Metadata.Penciler = mapContributor(w3cman.Penciler)
	manifest.Metadata.Narrator = mapContributor(w3cman.ReadBy)
	manifest.Metadata.Translator = mapContributor(w3cman.Translator)

	manifest.Links = mapLinks(w3cman.Links)
	manifest.ReadingOrder = mapLinks(w3cman.ReadingOrder)
	manifest.Resources = mapLinks(w3cman.Resources)

	// FIXME: add to the Readium manifest the ToC from index.html

	return
}

// BuildRPFFromLPF builds a Readium package (rwpp) from a W3C LPF file (lpfPath)
func BuildRPFFromLPF(lpfPath string, rwppPath string) error {

	// open the lpf file
	lpfFile, err := zip.OpenReader(lpfPath)
	if err != nil {
		return err
	}
	defer lpfFile.Close()

	// extract the W3C manifest from the LPF
	var w3cManifest rwpm.W3CPublication
	found := false
	for _, file := range lpfFile.File {
		if file.Name == W3CManifestName {
			m, err := file.Open()
			if err != nil {
				return err
			}
			defer m.Close()
			decoder := json.NewDecoder(m)
			err = decoder.Decode(&w3cManifest)
			if err != nil {
				return err
			}
			found = true
		}
	}
	// return an error if the W3C manifest missing
	if !found {
		return fmt.Errorf("W3C LPF %s: missing publication.json", lpfPath)
	}

	// extract the primary entry page from the LPF
	// FIXME: extract the primary entry page from the LPF

	// generate a Readium manifest out of the W3C manifest
	// and primary entry page
	rwpManifest := generateRWPManifest(w3cManifest)

	// marshal the Readium manifest
	rwpJSON, err := json.MarshalIndent(rwpManifest, "", " ")
	if err != nil {
		return err
	}
	// debug
	//println(string(rwpJSON))

	// create the rwpp file
	rwppFile, err := os.Create(rwppPath)
	defer rwppFile.Close()

	// create a zip writer on the rwpp
	zipWriter := zip.NewWriter(rwppFile)
	defer zipWriter.Close()

	// Add the Readium manifest to the rwpp
	man, err := zipWriter.Create(RWPManifestName)
	if err != nil {
		return err
	}
	_, err = man.Write(rwpJSON)

	// Append every lpf resource to the rwpp
	for _, file := range lpfFile.File {
		// filter MacOS specific files (present if a standard zipper has been used)
		runes := []rune(file.Name)
		if string(runes[:8]) == "__MACOSX" {
			continue
		}
		// keep the original compression value (store vs deflate)
		writer, err := zipWriter.CreateHeader(&file.FileHeader)
		// writer, err := zipWriter.Create(file.Name)
		if err != nil {
			return err
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	}
	return nil
}

// newUUID generates a random UUID according to RFC 4122
// note: this small function is copied from license.go
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// isoDurationToSc transforms an ISO duration to a number of seconds, as a float
func isoDurationToSc(iso string) (seconds float32, err error) {
	period, err := period.Parse(iso)
	seconds = float32(period.Hours()*3600 + period.Minutes()*60 + period.Seconds())
	return
}

// ------------------------- unused

// UnzipToFolder fills a folder (dest) with the content of a zip file (src)
// returns an array of unzipped file names
func UnzipToFolder(src string, dest string) ([]string, error) {

	var filepaths []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filepaths, err
	}
	defer r.Close()

	for _, f := range r.File {

		// store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filepaths, fmt.Errorf("%s: illegal file path", fpath)
		}

		filepaths = append(filepaths, fpath)

		if f.FileInfo().IsDir() {
			// make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filepaths, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filepaths, err
		}

		rc, err := f.Open()
		if err != nil {
			return filepaths, err
		}

		_, err = io.Copy(outFile, rc)

		// close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filepaths, err
		}
	}
	return filepaths, nil
}
