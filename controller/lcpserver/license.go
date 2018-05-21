/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package lcpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/controller/common"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/file_storage"
)

// GetLicense returns an existing license,
// selected by a license id and a partial license both given as input.
// The input partial license is optional: if absent, a partial license
// is returned to the caller, with the info stored in the db.
//
func GetLicense(resp http.ResponseWriter, req *http.Request, server common.IServer) {

	vars := mux.Vars(req)
	// get the license id from the request URL
	licenseID := vars["license_id"]

	//server.LogInfo("Get License with id : %s", licenseID)

	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusNotFound})
		return
	} else if e != nil {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusBadRequest})
		return
	}
	// get the input body.
	// It contains the hashed passphrase, user hint
	// and other optional user data the provider wants to see embedded in thel license
	licIn, err := common.ReadLicensePayload(req)
	// error parsing the input body
	if err != nil {
		// if there was no partial license given as payload, return a partial license.
		// The use case is a frontend that needs to get license up to date rights.
		if err.Error() == "EOF" {
			server.LogError("No payload, get a partial license")

			// add useful http headers
			resp.Header().Add(common.HdrContentType, common.ContentTypeLcpJson)
			resp.WriteHeader(http.StatusPartialContent)
			// send back the partial license
			// do not escape characters
			enc := json.NewEncoder(resp)
			enc.SetEscapeHTML(false)
			enc.Encode(licOut)
			return
		}
		// unknown error
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	// an input body was sent with the request:
	// check mandatory information in the partial license
	err = licIn.CheckGetLicenseInput()
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// copy useful data from licIn to LicOut
	licIn.CopyInputToLicense(licOut)
	// build the license
	err = buildLicense(licOut, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	// set the http headers
	resp.Header().Add(common.HdrContentType, common.ContentTypeLcpJson)
	resp.Header().Add(common.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	resp.WriteHeader(http.StatusOK)
	// send back the license
	// do not escape characters in the json payload
	enc := json.NewEncoder(resp)
	enc.SetEscapeHTML(false)
	enc.Encode(licOut)
}

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	// get the content id from the request URL
	contentID := vars["content_id"]

	//server.LogInfo("Generate License for content id : %v", contentID)

	// get the input body
	// note: no need to create licIn / licOut here, as the input body contains
	// info that we want to keep in the full license.

	lic, err := common.ReadLicensePayload(req)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	//server.LogInfo("Checking input.")
	// check mandatory information in the input body
	err = lic.CheckGenerateLicenseInput()
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	//server.LogInfo("Initialize.")
	// init the license with an id and issue date
	err = lic.Initialize(contentID)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	//server.LogInfo("Setting rights.")
	// normalize the start and end date, UTC, no milliseconds
	lic.SetRights()

	//server.LogInfo("Building license.")
	// build the license
	err = buildLicense(lic, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	//jsonPayload, err := json.MarshalIndent(lic, " ", " ")
	//server.LogInfo("Saving to database : %s", string(jsonPayload))
	// store the license in the db
	err = server.Store().License().Add(lic)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	// set http headers
	resp.Header().Add(common.HdrContentType, common.ContentTypeLcpJson)
	resp.Header().Add(common.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	resp.WriteHeader(http.StatusCreated)
	// send back the license
	// do not escape characters
	enc := json.NewEncoder(resp)
	enc.SetEscapeHTML(false)
	enc.Encode(lic)

	// notify the lsd server of the creation of the license.
	// this is an asynchronous call.
	go notifyLSDServer(lic, server)
}

// GetLicensedPublication returns a licensed publication
// for a given license identified by its id
// plus a partial license given as input
//
func GetLicensedPublication(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	licenseID := vars["license_id"]

	//server.LogInfo("Get a Licensed publication for license id : %s", licenseID)

	// get the input body
	licIn, err := common.ReadLicensePayload(req)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// check mandatory information in the input body
	err = licIn.CheckGetLicenseInput()
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusNotFound})
		return
	} else if e != nil {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusBadRequest})
		return
	}
	// copy useful data from licIn to LicOut
	licIn.CopyInputToLicense(licOut)
	// build the license
	err = buildLicense(licOut, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	// build a licenced publication
	publication, err := buildLicencedPublication(licOut, server)
	if err == file_storage.ErrNotFound {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Instance: licOut.ContentId, Status: http.StatusNotFound})
		return
	} else if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Instance: licOut.ContentId, Status: http.StatusInternalServerError})
		return
	}
	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := server.Store().Content().Get(licOut.ContentId)
	if err1 != nil {
		server.Error(resp, req, common.Problem{Detail: err1.Error(), Instance: licOut.ContentId, Status: http.StatusInternalServerError})
		return
	}
	location := content.Location

	// set HTTP headers
	resp.Header().Add(common.HdrContentType, epub.ContentTypeEpub)
	resp.Header().Add(common.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	resp.Header().Add(common.HdrXLcpLicense, licOut.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	resp.WriteHeader(http.StatusCreated)
	// return the full licensed publication to the caller
	publication.Write(resp)
}

// GenerateLicensedPublication generates and returns a licensed publication
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicensedPublication(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	contentID := vars["content_id"]

	//server.LogInfo("Generate a Licensed publication for content id : %s", contentID)

	// get the input body
	lic, err := common.ReadLicensePayload(req)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// check mandatory information in the input body
	err = lic.CheckGenerateLicenseInput()
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// init the license with an id and issue date
	err = lic.Initialize(contentID)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	// normalize the start and end date, UTC, no milliseconds
	lic.SetRights()
	// build the license
	err = buildLicense(lic, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	// store the license in the db
	err = server.Store().License().Add(lic)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Instance: contentID, Status: http.StatusInternalServerError})
		return
	}

	// notify the lsd server of the creation of the license
	go notifyLSDServer(lic, server)

	// build a licenced publication
	publication, err := buildLicencedPublication(lic, server)
	if err == file_storage.ErrNotFound {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Instance: lic.ContentId, Status: http.StatusNotFound})
		return
	} else if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Instance: lic.ContentId, Status: http.StatusInternalServerError})
		return
	}

	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := server.Store().Content().Get(lic.ContentId)
	if err1 != nil {
		server.Error(resp, req, common.Problem{Detail: err1.Error(), Instance: lic.ContentId, Status: http.StatusInternalServerError})
		return
	}
	location := content.Location

	// set HTTP headers
	resp.Header().Add(common.HdrContentType, epub.ContentTypeEpub)
	resp.Header().Add(common.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	resp.Header().Add(common.HdrXLcpLicense, lic.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	resp.WriteHeader(http.StatusCreated)
	// return the full licensed publication to the caller
	publication.Write(resp)
}

// UpdateLicense updates an existing license.
// parameters:
// 		{license_id} in the calling URL
// 		partial license containing properties which should be updated (and only these)
// return: an http status code (200, 400 or 404)
// Usually called from the License Status Server after a renew, return or cancel/revoke action
// -> updates the end date.
//
func UpdateLicense(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	// get the license id from the request URL
	licenseID := vars["license_id"]

	//server.LogInfo("Update License with id", licenseID)

	licIn, err := common.ReadLicensePayload(req)
	if err != nil { // no or incorrect (json) partial license found in the body
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusNotFound})
		return
	} else if e != nil {
		server.Error(resp, req, common.Problem{Detail: e.Error(), Status: http.StatusBadRequest})
		return
	}
	// update licOut using information found in licIn
	if licIn.User.UUID != "" {
		//server.LogInfo("new user id: %v", licIn.User.UUID)
		licOut.User.UUID = licIn.User.UUID
	}
	if licIn.Provider != "" {
		//server.LogInfo("new provider: %v", licIn.Provider)
		licOut.Provider = licIn.Provider
	}
	if licIn.ContentId != "" {
		//server.LogInfo("new content id: %v", licIn.ContentId)
		licOut.ContentId = licIn.ContentId
	}
	if licIn.Rights.Print.Valid {
		//server.LogInfo("new right, print: %v", licIn.Rights.Print)
		licOut.Rights.Print = licIn.Rights.Print
	}
	if licIn.Rights.Copy.Valid {
		//server.LogInfo("new right, copy: %v", licIn.Rights.Copy)
		licOut.Rights.Copy = licIn.Rights.Copy
	}
	if licIn.Rights.Start.Valid {
		//server.LogInfo("new right, start: %v", licIn.Rights.Start.Time)
		licOut.Rights.Start = licIn.Rights.Start
	}
	if licIn.Rights.End.Valid {
		//server.LogInfo("new right, end: %v", licIn.Rights.End.Time)
		licOut.Rights.End = licIn.Rights.End
	}
	// update the license in the database
	err = server.Store().License().Update(licOut)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
}

// ListLicenses returns a JSON struct with information about the existing licenses
// parameters:
// 	page: page number
//	per_page: number of items par page
//
func ListLicenses(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	var page int64
	var perPage int64
	var err error
	if req.FormValue("page") != "" {
		page, err = strconv.ParseInt((req).FormValue("page"), 10, 32)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		page = 1
	}
	if req.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((req).FormValue("per_page"), 10, 32)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		perPage = 30
	}
	if page > 0 { //pagenum starting at 0 in code, but user interface starting at 1
		page--
	}
	if page < 0 {
		server.Error(resp, req, common.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest})
		return
	}

	licenses, err := server.Store().License().ListAll(int(perPage), int(page))
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		resp.Header().Set("LicenseLink", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		resp.Header().Set("LicenseLink", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	resp.Header().Set(common.HdrContentType, common.ContentTypeJson)

	enc := json.NewEncoder(resp)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(licenses)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
}

// ListLicensesForContent lists all licenses associated with a given content
// parameters:
//	content_id: content identifier
// 	page: page number (default 1)
//	per_page: number of items par page (default 30)
//
func ListLicensesForContent(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	var page int64
	var perPage int64
	var err error
	contentID := vars["content_id"]

	//check if the license exists
	_, err = server.Store().Content().Get(contentID)
	if err == gorm.ErrRecordNotFound {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusNotFound})
		return
	} //other errors pass, but will probably reoccur
	if req.FormValue("page") != "" {
		page, err = strconv.ParseInt(req.FormValue("page"), 10, 32)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		page = 1
	}

	if req.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((req).FormValue("per_page"), 10, 32)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		perPage = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		server.Error(resp, req, common.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest})
		return
	}

	licenses, err := server.Store().License().List(contentID, int(perPage), int(page))
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
	}

	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		resp.Header().Set("LicenseLink", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		resp.Header().Set("LicenseLink", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	resp.Header().Set(common.HdrContentType, common.ContentTypeJson)
	enc := json.NewEncoder(resp)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(licenses)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

}
