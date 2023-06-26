// Copyright 2017 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
)

// GetLicenseHandler returns an existing license,
// selected by a license id and a partial license both given as input.
// The input partial license is optional: if absent, a partial license
// is returned to the caller, with the info stored in the db.
func GetLicenseHandler(w http.ResponseWriter, r *http.Request, s Server) {
	// get the input body.
	// It contains the hashed passphrase, user hint
	// and other optional user data the provider wants to see embedded in the license
	var licIn license.License
	err := DecodeJSONLicense(r, &licIn)
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
			writeResponseLicense(w, r, &licIn, s)
			return
		}
		// unknown error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// set the http headers
	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
	w.WriteHeader(http.StatusOK)

	writeResponseLicense(w, r, &licIn, s)
}

func writeResponseLicense(w http.ResponseWriter, r *http.Request, licIn *license.License, s Server) {
	// send back the license
	// do not escape characters in the json payload
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	licOut := getExistingLicense(w, r, licIn, s)
	if licOut != nil {
		err := enc.Encode(licOut)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
}

func getExistingLicense(w http.ResponseWriter, r *http.Request, licIn *license.License, s Server) *license.License {
	vars := mux.Vars(r)
	// get the license id from the request URL
	licenseID := vars["license_id"]

	log.Println("Get License with id", licenseID)
	// initialize the license from the info stored in the db.
	licOut, err := GetLicense(licenseID, licIn, s)
	// process license not found etc.
	if err == license.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		return nil
	} else if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return nil
	}
	return licOut
}

// GenerateLicenseHandler generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
func GenerateLicenseHandler(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)
	// get the content id from the request URL
	contentID := vars["content_id"]

	// get the input body
	// note: no need to create licIn / licOut here, as the input body contains
	// info that we want to keep in the full license.
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	err = GenerateLicense(contentID, &lic, s)
	if _, ok := err.(*ErrBadLicenseInput); ok {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
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
}

// GetLicensedPublication returns a licensed publication
// for a given license identified by its id
// plus a partial license given as input
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

	licOut, err := GetLicense(licenseID, &licIn, s)
	if _, ok := err.(*ErrBadLicenseInput); ok {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	// TODO: Licenses().get(licenseID) failures
	// process license not found etc.
	/*if e == license.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		return
	} else if e != nil {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		return
	}*/

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// build a licensed publication
	err = writeLicensedPublication(licOut, w, s)
	if err == storage.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: licOut.ContentID}, http.StatusNotFound)
		return
	} else if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: licOut.ContentID}, http.StatusInternalServerError)
		return
	}
}

// GenerateLicensedPublication generates and returns a licensed publication
// for a given content identified by its id
// plus a partial license given as input
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

	err = GenerateLicense(contentID, &lic, s)
	if _, ok := err.(*ErrBadLicenseInput); ok {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	// notify the lsd server of the creation of the license
	go notifyLsdServer(lic, s)

	err = writeLicensedPublication(&lic, w, s)
	if err == storage.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: lic.ContentID}, http.StatusNotFound)
		return
	} else if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: lic.ContentID}, http.StatusInternalServerError)
		return
	}
}

func writeLicensedPublication(lic *license.License, w http.ResponseWriter, s Server) error {
	// build a licenced publication
	buf, location, err := BuildLicensedPublication(lic, s)
	if err != nil {
		return err
	}
	// set HTTP headers
	w.Header().Add("Content-Type", epub.ContentType_EPUB)
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	w.Header().Add("X-Lcp-License", lic.ID)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
	// return the full licensed publication to the caller
	io.Copy(w, &buf)
	return nil
}

// UpdateLicense updates an existing license.
// parameters:
//
//	{license_id} in the calling URL
//	partial license containing properties which should be updated (and only these)
//
// return: an http status code (200, 400 or 404)
// Usually called from the License Status Server after a renew, return or cancel/revoke action
// -> updates the end date.
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
	if e == license.ErrNotFound {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		return
	} else if e != nil {
		problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		return
	}
	// update licOut using information found in licIn
	if licIn.User.ID != "" {
		log.Println("new user id: ", licIn.User.ID)
		licOut.User.ID = licIn.User.ID
	}
	if licIn.Provider != "" {
		log.Println("new provider: ", licIn.Provider)
		licOut.Provider = licIn.Provider
	}
	if licIn.ContentID != "" {
		log.Println("new content id: ", licIn.ContentID)
		licOut.ContentID = licIn.ContentID
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
//
//	page: page number
//	per_page: number of items par page
func ListLicenses(w http.ResponseWriter, r *http.Request, s Server) {

	var page int64
	var perPage int64
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
		perPage, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		perPage = 30
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
	fn := s.Licenses().ListAll(int(perPage), int(page))
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
//
//	content_id: content identifier
//	page: page number (default 1)
//	per_page: number of items par page (default 30)
func ListLicensesForContent(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)
	var page int64
	var perPage int64
	var err error
	contentID := vars["content_id"]

	// check if the license exists
	_, err = s.Index().Get(contentID)
	if err == index.ErrNotFound {
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
		perPage, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		perPage = 30
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
	fn := s.Licenses().ListByContentID(contentID, int(perPage), int(page))
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
