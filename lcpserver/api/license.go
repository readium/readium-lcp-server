// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilcp

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/sign"
	"github.com/readium/readium-lcp-server/storage"
)

// GetLicense returns a license,
// selected by a license id and a partial license both given as input
// the input partial license is optional: if absent, a partial license
// is returned to the caller.
//
func GetLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenceID := vars["license_id"]

	// get the license from the db, by its id
	var aLicense license.License
	aLicense, e := s.Licenses().Get(licenceID)
	if e != nil {
		if e == license.NotFound {
			problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		}
		return
	}
	// get the partial license given as input
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil {
		// no partial license as payload
		if err.Error() == "EOF" {

			log.Println("No payload, get a partial license")

			err = prepareLinks(aLicense, s)
			if err != nil {
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
			// partial content
			w.WriteHeader(http.StatusPartialContent)

			//delete some sensitive data from license
			aLicense.Encryption.UserKey.Check = nil
			aLicense.Encryption.UserKey.Value = nil
			aLicense.Encryption.UserKey.Hint = ""
			aLicense.Encryption.UserKey.ClearValue = ""
			aLicense.Encryption.UserKey.Key.Algorithm = ""
			aLicense.Encryption.Profile = ""
			// return the partial license
			enc := json.NewEncoder(w)
			// does not escape characters
			enc.SetEscapeHTML(false)
			enc.Encode(aLicense)
			return
		}
		// unknow error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// debug message, in case the input license causes problems
	/*
		log.Print("Partial license:")
		jsonBody, err := json.Marshal(lic)
		if err != nil {
			return
		}
		log.Print(string(jsonBody))
	*/

	// check that the user id is available
	if lic.User.Id == "" {
		problem.Error(w, r, problem.Problem{Detail: "User identification must be available"}, http.StatusBadRequest)
		return
	}
	// set user information in the license
	aLicense.User = lic.User
	// set
	if aLicense.Links == nil {
		aLicense.Links = license.DefaultLinksCopy()
	}
	// set the passphrase hash in the license for later use
	aLicense.Encryption.UserKey.Value = lic.Encryption.UserKey.Value

	// add information to the license, sign
	err = completeLicense(&aLicense, aLicense.ContentId, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// set the http headers
	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
	// note: must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusOK)
	// return the license
	enc := json.NewEncoder(w)
	// does not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(aLicense)
}

// updateLicenseInDatabase updates the license in the database
// parameters:
//	 partial license, containing any of:
//		 provider
//		 usage rights: start and end dates, copy and print
//		 passphrase hint
//		 content id
// return: a partial license with updated properties
//
func updateLicenseInDatabase(licenseID string, partialLicense license.License, s Server) (license.License, error) {

	var aLicense license.License
	aLicense, err := s.Licenses().Get(licenseID)
	if err != nil {
		return aLicense, err
	}

	// update rights of license in database / verify validity of lic / existingLicense
	if partialLicense.Provider != "" {
		aLicense.Provider = partialLicense.Provider
	}
	if partialLicense.Rights != nil {
		if partialLicense.Rights.Copy != nil {
			aLicense.Rights.Copy = partialLicense.Rights.Copy
		}
		if partialLicense.Rights.Print != nil {
			aLicense.Rights.Print = partialLicense.Rights.Print
		}
		if partialLicense.Rights.Start != nil {
			aLicense.Rights.Start = partialLicense.Rights.Start
		}
		if partialLicense.Rights.End != nil {
			aLicense.Rights.End = partialLicense.Rights.End
		}
	} else {
		aLicense.Rights.Copy = nil
		aLicense.Rights.Print = nil
		aLicense.Rights.Start = nil
		aLicense.Rights.End = nil
	}

	if partialLicense.Encryption.UserKey.Hint != "" {
		aLicense.Encryption.UserKey.Hint = partialLicense.Encryption.UserKey.Hint
	}
	if partialLicense.ContentId != "" {
		aLicense.ContentId = partialLicense.ContentId
	}
	err = s.Licenses().Update(aLicense)

	return aLicense, err
}

// UpdateLicense updates an existing license.
// parameters:
// 		{license_id} in the calling URL
// 		partial license containing properties which should be updated (and only these)
// return: an http status code (200, 400 or 404)
// Usually called from the License Status Server after a renew, return or cancel/revoke action
// -> updates the end date.
//
func UpdateLicense(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)
	licenseID := vars["license_id"]
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil { // no or incorrect (json) partial license found in the body
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// FIXME: no use for that, remove the id in the payload.
	if lic.Id != licenseID {
		problem.Error(w, r, problem.Problem{Detail: "Different license IDs"}, http.StatusNotFound)
		return
	}
	// update the license in the database
	_, err = updateLicenseInDatabase(licenseID, lic, s)
	if err != nil {
		if err == license.NotFound {
			problem.Error(w, r, problem.Problem{Detail: license.NotFound.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		}
		return
	}
	// go on and GET license io to return the updated license
	// must pass a partial license (id + rights)
	// FIXME: does not return a partial license, only a 200 ok. The doc must be modified.
	// GetLicense(w, r, s)
}

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var lic license.License

	err := DecodeJSONLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	// normalize the start and end date, UTC, no milliseconds
	// start/ end may be null
	var start, end time.Time
	if lic.Rights.Start != nil {
		start = lic.Rights.Start.UTC().Truncate(time.Second)
		lic.Rights.Start = &start
	}
	if lic.Rights.End != nil {
		end = lic.Rights.End.UTC().Truncate(time.Second)
		lic.Rights.End = &end
	}

	contentID := vars["content_id"]
	lic.ContentId = ""
	err = completeLicense(&lic, contentID, s)

	if err != nil {
		if err == storage.ErrNotFound || err == index.NotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}

	// store the license info in the db
	err = s.Licenses().Add(lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	// does not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(lic)
}

// GenerateProtectedPublication generates and returns a protected publication
// for a given license identified by its id
// or
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateProtectedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	var partialLicense license.License
	var newLicense license.License
	err := DecodeJSONLicense(r, &partialLicense)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	contentID := vars["content_id"] // may be empty
	licenseID := vars["license_id"] // may be empty

	if licenseID != "" { // POST /{license_id}/publication
		//license update, regenerate publication, maybe only get from db ?
		newLicense, err = updateLicenseInDatabase(licenseID, partialLicense, s)
		if err != nil {
			if err == license.NotFound {
				problem.Error(w, r, problem.Problem{Detail: license.NotFound.Error()}, http.StatusNotFound)
			} else {
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			}
			return
		}
		newLicense.User = partialLicense.User //pass user information in updated license
		// set the passphrase hash in the license for later use
		newLicense.Encryption.UserKey.Value = partialLicense.Encryption.UserKey.Value
		// FIXME: remove clear value
		//newLicense.Encryption.UserKey.ClearValue = partialLicense.Encryption.UserKey.ClearValue

		// contentID is not set, get it from the license
		contentID = newLicense.ContentId
		err = completeLicense(&newLicense, contentID, s)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}

	} else { //	 POST //{content_id}/publication[s]
		//new license, generate publication
		newLicense = partialLicense
		newLicense.ContentId = ""
		err = completeLicense(&newLicense, contentID, s)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}
		err = s.Licenses().Add(newLicense)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}
		licenseID = newLicense.Id
	}

	// get the publication, add the license to the publication
	if contentID == "" {
		problem.Error(w, r, problem.Problem{Detail: "No content id", Instance: contentID}, http.StatusBadRequest)
		return
	}

	epubFile, err := s.Store().Get(contentID)
	if err != nil {
		if err == storage.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	content, err := s.Index().Get(contentID)
	if err != nil {
		if err == index.NotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}
	var b bytes.Buffer
	contents, err := epubFile.Contents()
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	io.Copy(&b, contents)
	zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	// add the license to publication
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(newLicense)
	// Suppress the trailing newline
	// FIXME/ try to optimize with buf.ReadBytes(byte('\n')) instead of creating a new buffer.
	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add(epub.LicenseFile, &buf2, uint64(buf2.Len()))

	// set HTTP headers
	w.Header().Add("Content-Type", epub.ContentType_EPUB)
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	w.Header().Add("X-Lcp-License", newLicense.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
	// write HTTP body
	ep.Write(w)
}

// DecodeJSONLicense decodes a license formatted in json and returns a license object
//
func DecodeJSONLicense(r *http.Request, lic *license.License) error {
	var dec *json.Decoder

	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_FORM_URL_ENCODED {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	err := dec.Decode(&lic)

	return err
}

// completeLicense generates a complete license out of a partial license
//
func completeLicense(l *license.License, contentID string, s Server) error {
	c, err := s.Index().Get(contentID)
	if err != nil {
		return err
	}

	isNewLicense := l.ContentId == ""
	if isNewLicense {
		license.Prepare(l)
		l.ContentId = contentID
	} else {
		l.Signature = nil // empty signature fields, needs to be recalculated
	}
	links := new([]license.Link)

	// verify that mandatory hint link is present in the input license
	if value, present := license.DefaultLinks["hint"]; present {
		hint := license.Link{Href: value, Rel: "hint"}
		*links = append(*links, hint)
	} else {
		return errors.New("No hint link present in the config file")
	}

	// verify that mandatory publication link is present in the input license
	// replace the publication_id template by the actual value
	if value, present := license.DefaultLinks["publication"]; present {
		if !strings.Contains(value, "{publication_id}") {
			return errors.New("Missing {publication_id} template in publication link in config file")
		}
		// replace {publication_id} in template link
		publicationLink := strings.Replace(value, "{publication_id}", c.Id, 1)
		publication := license.Link{Href: publicationLink, Rel: "publication", Type: epub.ContentType_EPUB, Size: c.Length, Title: c.Location, Checksum: c.Sha256}
		*links = append(*links, publication)
	} else {
		return errors.New("Missing publication link in config file")
	}

	// verify that the mandatory status link is present in the configuration
	// note that the status link is made mandatory in the production lcp ecosystem
	// replace the license_id template by the actual value
	if value, present := config.Config.License.Links["status"]; present { // add status server to License
		if !strings.Contains(value, "{license_id}") {
			return errors.New("Missing {license_id} template in status link in config file")
		}
		statusLink := strings.Replace(value, "{license_id}", l.Id, 1)

		status := license.Link{Href: statusLink, Rel: "status", Type: api.ContentType_LSD_JSON} //status.Type = ??
		*links = append(*links, status)
	} else {
		return errors.New("Missing status link in config file")
	}

	l.Links = *links

	// generate the user key
	encryptionKey := license.GenerateUserKey(l.Encryption.UserKey)

	// empty the passphrase hash to avoid sending it to the user
	l.Encryption.UserKey.Value = nil

	// encrypt content key with user key
	encrypterContentKey := crypto.NewAESEncrypter_CONTENT_KEY()

	l.Encryption.ContentKey.Algorithm = encrypterContentKey.Signature()
	l.Encryption.ContentKey.Value = encryptKey(encrypterContentKey, c.EncryptionKey, encryptionKey[:])
	l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"

	encrypterFields := crypto.NewAESEncrypter_FIELDS()

	err = encryptFields(encrypterFields, l, encryptionKey[:])
	if err != nil {
		return err
	}
	// build the key check
	encrypterUserKeyCheck := crypto.NewAESEncrypter_USER_KEY_CHECK()

	err = buildKeyCheck(encrypterUserKeyCheck, l, encryptionKey[:])
	if err != nil {
		return err
	}
	// sign the license
	if l.Signature != nil {
		log.Println("Signature is NOT nil (it should)")
		l.Signature = nil
	}
	err = signLicense(l, s.Certificate())
	if err != nil {
		return err
	}
	return nil
}

// buildKeyCheck
//
func buildKeyCheck(encrypter crypto.Encrypter, l *license.License, key []byte) error {
	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(l.Id), &out)
	if err != nil {
		return err
	}
	l.Encryption.UserKey.Check = out.Bytes()
	return nil
}

// encryptFields
//
func encryptFields(encrypter crypto.Encrypter, l *license.License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(&l.User, toEncrypt)
		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

// getField
//
func getField(u *license.UserInfo, field string) reflect.Value {
	v := reflect.ValueOf(u).Elem()
	return v.FieldByName(strings.Title(field))
}

// signLicense
//
func signLicense(l *license.License, cert *tls.Certificate) error {
	sig, err := sign.NewSigner(cert)
	if err != nil {
		return err
	}
	res, err := sig.Sign(l)
	if err != nil {
		return err
	}
	l.Signature = &res

	return nil
}

// encryptKey
//
func encryptKey(encrypter crypto.Encrypter, key []byte, kek []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(key)
	encrypter.Encrypt(kek[:], in, &out)
	return out.Bytes()
}

// ListLicenses returns a JSON struct with information about the existing licenses
// parameters:
// 	page: page number
//	per_page: number of items par page
//
func ListLicenses(w http.ResponseWriter, r *http.Request, s Server) {
	var page int64
	var per_page int64
	var err error
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}
	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 { //pagenum starting at 0 in code, but user interface starting at 1
		page--
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	licenses := make([]license.LicenseReport, 0)
	//log.Println("ListAll(" + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.Licenses().ListAll(int(per_page), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// ListLicensesForContent lists all licenses associated with a given content
// parameters:
//	content_id: content identifier
// 	page: page number (default 1)
//	per_page: number of items par page (default 30)
//
func ListLicensesForContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var page int64
	var per_page int64
	var err error
	contentId := vars["content_id"]
	//check if license exists
	_, err = s.Index().Get(contentId)
	if err == index.NotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		return
	} //other errors pass, but will probably reoccur
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt(r.FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}

	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	licenses := make([]license.LicenseReport, 0)
	//log.Println("List(" + contentId + "," + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.Licenses().List(contentId, int(per_page), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

// prepareLinks
//
func prepareLinks(license license.License, s Server) error {
	for i := 0; i < len(license.Links); i++ {
		if license.Links[i].Rel == "publication" {
			item, err := s.Index().Get(license.ContentId)
			if err != nil {
				return err
			}
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{publication_id}", license.ContentId, 1)
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{publication_loc}", item.Location, 1)
		}

		if license.Links[i].Rel == "status" {
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{license_id}", license.Id, 1)
		}
	}
	return nil
}
