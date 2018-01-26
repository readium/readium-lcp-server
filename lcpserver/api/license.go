// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilcp

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
)

// ErrMandatoryInfoMissing sets an error message returned to the caller
var ErrMandatoryInfoMissing = errors.New("Mandatory info missing in the input body")

// get license, check mandatory information in the input body
//
func checkGetLicenseInput(l *license.License) error {
	if l.Encryption.UserKey.Hint == "" {
		log.Println("User hint is missing")
		return ErrMandatoryInfoMissing
	}
	if l.Encryption.UserKey.Value == nil {
		log.Println("User hashed passphrase is missing")
		return ErrMandatoryInfoMissing
	}
	if l.Encryption.UserKey.Algorithm == "" {
		log.Println("User passphrase hash algorithm is missing, set default value")
		// the only valid value in LCP basic and 10 profiles is sha256
		l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	}
	return nil
}

// generate license, check mandatory information in the input body
//
func checkGenerateLicenseInput(l *license.License) error {
	if l.Provider == "" {
		log.Println("License provider is missing")
		return ErrMandatoryInfoMissing
	}
	if l.User.Id == "" {
		log.Println("User identification is missing")
		return ErrMandatoryInfoMissing
	}
	// check userkey hint, value and algorithm
	err := checkGetLicenseInput(l)
	return err
}

// get license, copy useful data from licIn to LicOut
//
func copyInputToLicense(licIn *license.License, licOut *license.License) {
	// copy the hashed passphrase, user hint and algorithm
	licOut.Encryption.UserKey = licIn.Encryption.UserKey
	// copy optional user information
	licOut.User.Email = licIn.User.Email
	licOut.User.Name = licIn.User.Name
	licOut.User.Encrypted = licIn.User.Encrypted
}

// normalize the start and end date, UTC, no milliseconds
//
func setRights(lic *license.License) {
	// a rights object is needed before adding a record to the db
	if lic.Rights == nil {
		lic.Rights = new(license.UserRights)
	}
	if lic.Rights.Start != nil {
		start := lic.Rights.Start.UTC().Truncate(time.Second)
		lic.Rights.Start = &start
	}
	if lic.Rights.End != nil {
		end := lic.Rights.End.UTC().Truncate(time.Second)
		lic.Rights.End = &end
	}
}

// build a license, common to get and generate license, get and generate licensed publication
//
func buildLicense(lic *license.License, s Server) error {

	// set the LCP profile
	license.SetLicenseProfile(lic)

	// get content info from the db
	content, err := s.Index().Get(lic.ContentId)
	if err != nil {
		log.Println("No content with id", lic.ContentId)
		return err
	}

	// set links
	err = license.SetLicenseLinks(lic, content)
	if err != nil {
		return err
	}
	// encrypt the content key, user fieds, set the key check
	err = license.EncryptLicenseFields(lic, content)
	if err != nil {
		return err
	}
	// sign the license
	err = license.SignLicense(lic, s.Certificate())
	if err != nil {
		return err
	}
	return nil
}

// build a licensed publication, common to get and generate licensed publication
//
func buildLicencedPublication(lic *license.License, s Server) (ep epub.Epub, err error) {
	// get the epub content info from the bd
	epubFile, err := s.Store().Get(lic.ContentId)
	if err != nil {
		return
	}
	// get the epub content
	epubContent, err1 := epubFile.Contents()
	if err1 != nil {
		return ep, err1
	}
	var b bytes.Buffer
	// copy the epub content to a buffer
	io.Copy(&b, epubContent)
	// create a zip reader
	zr, err2 := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err2 != nil {
		return ep, err2
	}
	ep, err3 := epub.Read(zr)
	if err3 != nil {
		return ep, err3
	}
	// add the license to publication
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(lic)
	// write the buffer in the zip, and suppress the trailing newline
	// FIXME: check that the newline is not present anymore
	// FIXME/ try to optimize with buf.ReadBytes(byte('\n')) instead of creating a new buffer.
	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add(epub.LicenseFile, &buf2, uint64(buf2.Len()))
	return
}

// GetLicense returns an existing license,
// selected by a license id and a partial license both given as input.
// The input partial license is optional: if absent, a partial license
// is returned to the caller, with the info stored in the db.
//
func GetLicense(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)
	// get the license id from the request URL
	licenseID := vars["license_id"]

	log.Println("Get License with id", licenseID)

	// initialize the license from the info stored in the db.
	var licOut license.License
	licOut, e := s.Licenses().Get(licenseID)
	// process license not found etc.
	if e == license.NotFound {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		return
	} else if e != nil {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		return
	}
	// get the input body.
	// It contains the hashed passphrase, user hint
	// and other optional user data the provider wants to see embedded in thel license
	var err error
	var licIn license.License
	err = DecodeJSONLicense(r, &licIn)
	// error parsing the input body
	if err != nil {
		// if there was no partial license given as payload, return a partial license.
		// The use case is a frontend that needs to get license up to date rights.
		if err.Error() == "EOF" {
			log.Println("No payload, get a partial license")

			// add useful http headers
			w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
			w.WriteHeader(http.StatusPartialContent)
			// send back the partial license
			// do not escape characters
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			enc.Encode(licOut)
			return
		}
		// unknown error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// an input body was sent with the request:
	// check mandatory information in the partial license
	err = checkGetLicenseInput(&licIn)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// copy useful data from licIn to LicOut
	copyInputToLicense(&licIn, &licOut)
	// build the license
	err = buildLicense(&licOut, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// set the http headers
	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
	w.WriteHeader(http.StatusOK)
	// send back the license
	// do not escape characters in the json payload
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(licOut)
}

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	// get the content id from the request URL
	contentID := vars["content_id"]

	log.Println("Generate License for content id", contentID)

	// get the input body
	// note: no need to create licIn / licOut here, as the input body contains
	// info that we want to keep in the full license.
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// check mandatory information in the input body
	err = checkGenerateLicenseInput(&lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// init the license with an id and issue date
	license.Initialize(contentID, &lic)

	// normalize the start and end date, UTC, no milliseconds
	setRights(&lic)

	// build the license
	err = buildLicense(&lic, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// store the license in the db
	err = s.Licenses().Add(lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		//problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	// set http headers
	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
	w.WriteHeader(http.StatusCreated)
	// send back the license
	// do not escape characters
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(lic)

	// notify the lsd server of the creation of the license.
	// this is an asynchronous call.
	go notifyLsdServer(lic, s)
}

// GetLicensedPublication returns a licensed publication
// for a given license identified by its id
// plus a partial license given as input
//
func GetLicensedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	licenseID := vars["license_id"]

	log.Println("Get a Licensed publication for license id", licenseID)

	// get the input body
	var licIn license.License
	err := DecodeJSONLicense(r, &licIn)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// check mandatory information in the input body
	err = checkGetLicenseInput(&licIn)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// initialize the license from the info stored in the db.
	licOut, e := s.Licenses().Get(licenseID)
	// process license not found etc.
	if e == license.NotFound {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		return
	} else if e != nil {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		return
	}
	// copy useful data from licIn to LicOut
	copyInputToLicense(&licIn, &licOut)
	// build the license
	err = buildLicense(&licOut, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// build a licenced publication
	var ep epub.Epub
	ep, err = buildLicencedPublication(&licOut, s)
	if err == storage.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: licOut.ContentId}, http.StatusNotFound)
		return
	} else if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: licOut.ContentId}, http.StatusInternalServerError)
		return
	}
	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := s.Index().Get(licOut.ContentId)
	if err1 != nil {
		problem.Error(w, r, problem.Problem{Detail: err1.Error(), Instance: licOut.ContentId}, http.StatusInternalServerError)
		return
	}
	location := content.Location

	// set HTTP headers
	w.Header().Add("Content-Type", epub.ContentType_EPUB)
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	w.Header().Add("X-Lcp-License", licOut.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
	// return the full licensed publication to the caller
	ep.Write(w)
}

// GenerateLicensedPublication generates and returns a licensed publication
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicensedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	contentID := vars["content_id"]

	log.Println("Generate a Licensed publication for content id", contentID)

	// get the input body
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// check mandatory information in the input body
	err = checkGenerateLicenseInput(&lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// init the license with an id and issue date
	license.Initialize(contentID, &lic)
	// normalize the start and end date, UTC, no milliseconds
	setRights(&lic)
	// build the license
	err = buildLicense(&lic, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// store the license in the db
	err = s.Licenses().Add(lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	// notify the lsd server of the creation of the license
	go notifyLsdServer(lic, s)

	// build a licenced publication
	var ep epub.Epub
	ep, err = buildLicencedPublication(&lic, s)
	if err == storage.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: lic.ContentId}, http.StatusNotFound)
		return
	} else if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: lic.ContentId}, http.StatusInternalServerError)
		return
	}

	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := s.Index().Get(lic.ContentId)
	if err1 != nil {
		problem.Error(w, r, problem.Problem{Detail: err1.Error(), Instance: lic.ContentId}, http.StatusInternalServerError)
		return
	}
	location := content.Location

	// set HTTP headers
	w.Header().Add("Content-Type", epub.ContentType_EPUB)
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	w.Header().Add("X-Lcp-License", lic.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
	// return the full licensed publication to the caller
	ep.Write(w)
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
	// get the license id from the request URL
	licenseID := vars["license_id"]

	log.Println("Update License with id", licenseID)

	var licIn license.License
	err := DecodeJSONLicense(r, &licIn)
	if err != nil { // no or incorrect (json) partial license found in the body
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// initialize the license from the info stored in the db.
	var licOut license.License
	licOut, e := s.Licenses().Get(licenseID)
	// process license not found etc.
	if e == license.NotFound {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		return
	} else if e != nil {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		return
	}
	// update licOut using information found in licIn
	if licIn.User.Id != "" {
		log.Println("new user id: ", licIn.User.Id)
		licOut.User.Id = licIn.User.Id
	}
	if licIn.Provider != "" {
		log.Println("new provider: ", licIn.Provider)
		licOut.Provider = licIn.Provider
	}
	if licIn.ContentId != "" {
		log.Println("new content id: ", licIn.ContentId)
		licOut.ContentId = licIn.ContentId
	}
	if licIn.Rights.Print != nil {
		log.Println("new right, print: ", *licIn.Rights.Print)
		licOut.Rights.Print = licIn.Rights.Print
	}
	if licIn.Rights.Copy != nil {
		log.Println("new right, copy: ", *licIn.Rights.Copy)
		licOut.Rights.Copy = licIn.Rights.Copy
	}
	if licIn.Rights.Start != nil {
		log.Println("new right, start: ", *licIn.Rights.Start)
		licOut.Rights.Start = licIn.Rights.Start
	}
	if licIn.Rights.End != nil {
		log.Println("new right, end: ", *licIn.Rights.End)
		licOut.Rights.End = licIn.Rights.End
	}
	// update the license in the database
	err = s.Licenses().Update(licOut)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
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
	contentID := vars["content_id"]

	//check if the license exists
	_, err = s.Index().Get(contentID)
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
	fn := s.Licenses().List(contentID, int(per_page), int(page))
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

	if err != nil && err.Error() != "EOF" {
		log.Print("Decode license: invalid json structure")
	}
	return err
}

// notifyLsdServer informs the License Status Server of the creation of a new license
// and saves the result of the http request in the DB (using *Store)
//
func notifyLsdServer(l license.License, s Server) {
	if config.Config.LsdServer.PublicBaseUrl != "" {
		var lsdClient = &http.Client{
			Timeout: time.Second * 10,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_ = json.NewEncoder(pw).Encode(l)
			pw.Close() // signal end writing
		}()
		req, err := http.NewRequest("PUT", config.Config.LsdServer.PublicBaseUrl+"/licenses", pr)
		if err != nil {
			return
		}
		// set credentials on lsd request
		notifyAuth := config.Config.LsdNotifyAuth
		if notifyAuth.Username != "" {
			req.SetBasicAuth(notifyAuth.Username, notifyAuth.Password)
		}

		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

		response, err := lsdClient.Do(req)
		if err != nil {
			log.Println("Error Notify LsdServer of new License (" + l.Id + "):" + err.Error())
			_ = s.Licenses().UpdateLsdStatus(l.Id, -1)
		} else {
			defer req.Body.Close()
			_ = s.Licenses().UpdateLsdStatus(l.Id, int32(response.StatusCode))
			// message to the console
			log.Println("Notify Lsd Server of a new License with id " + l.Id + " = " + strconv.Itoa(response.StatusCode))
		}
	}
}

// utility: log a license for debug purposes
// ex: logLicense("build licence:", licOut)

func logLicense(msg string, l *license.License) {
	log.Println(msg)
	jsonBody, errj := json.Marshal(*l)
	if errj != nil {
		log.Println("logLicense: not well formed json")
		return
	}
	log.Print(string(jsonBody))
}
