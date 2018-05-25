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
	"strconv"

	"bytes"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
)

// GetLicense returns an existing license,
// selected by a license id and a partial license both given as input.
// The input partial license is optional: if absent, a partial license
// is returned to the caller, with the info stored in the db.
//
func GetLicense(server http.IServer, licIn *model.License, id ParamLicenseId) (*model.License, error) {
	if id.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}
	licenseID := id.LicenseID
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusNotFound}
	} else if e != nil {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusBadRequest}
	}
	// TODO : below is not working anymore
	/**
		// get the input body.
		// It contains the hashed passphrase, user hint
		// and other optional user data the provider wants to see embedded in thel license
		licIn, err := ReadLicensePayload(req)
		// error parsing the input body
		if err != nil {
			// if there was no partial license given as payload, return a partial license.
			// The use case is a frontend that needs to get license up to date rights.
			if err.Error() == "EOF" {
				server.LogError("No payload, get a partial license")

				// add useful http headers
				resp.Header().Add(http.HdrContentType, http.ContentTypeLcpJson)
				resp.WriteHeader(http.StatusPartialContent)
				// send back the partial license
				// do not escape characters
				enc := json.NewEncoder(resp)
				enc.SetEscapeHTML(false)
				enc.Encode(licOut)
				return licOut, nil
			}
			// unknown error
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	**/
	// an input body was sent with the request:
	// check mandatory information in the partial license
	err := licIn.CheckGetLicenseInput()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// copy useful data from licIn to LicOut
	licIn.CopyInputToLicense(licOut)
	// build the license
	err = buildLicense(licOut, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// TODO : set below headers
	// set the http headers
	//resp.Header().Add(http.HdrContentType, http.ContentTypeLcpJson)
	//resp.Header().Add(http.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	//resp.WriteHeader(http.StatusOK)
	// send back the license
	// TODO : do not escape characters in the json payload
	return licOut, nil
}

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(server http.IServer, lic *model.License, param ParamContentId) (*model.License, error) {
	if param.ContentID == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}
	contentID := param.ContentID
	//server.LogInfo("Checking input.")
	// check mandatory information in the input body
	err := lic.CheckGenerateLicenseInput()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	//server.LogInfo("Initialize.")
	// init the license with an id and issue date
	err = lic.Initialize(contentID)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	//server.LogInfo("Setting rights.")
	// normalize the start and end date, UTC, no milliseconds
	lic.SetRights()

	//server.LogInfo("Building license.")
	// build the license
	err = buildLicense(lic, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	//jsonPayload, err := json.MarshalIndent(lic, " ", " ")
	//server.LogInfo("Saving to database : %s", string(jsonPayload))
	// store the license in the db
	err = server.Store().License().Add(lic)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// TODO : set below headers
	// set http headers
	//resp.Header().Add(http.HdrContentType, http.ContentTypeLcpJson)
	//resp.Header().Add(http.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	//resp.WriteHeader(http.StatusCreated)
	// TODO : do not escape characters
	//	send back the license
	//enc := json.NewEncoder(resp)
	//enc.SetEscapeHTML(false)
	//enc.Encode(lic)

	// notify the lsd server of the creation of the license.
	// this is an asynchronous call.
	go notifyLSDServer(lic, server)
	return lic, nil
}

// GetLicensedPublication returns a licensed publication
// for a given license identified by its id
// plus a partial license given as input
//
func GetLicensedPublication(server http.IServer, licIn *model.License, id ParamLicenseId) (*epub.Epub, error) {
	if id.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}
	licenseID := id.LicenseID
	// get the input body
	//licIn, err := ReadLicensePayload(req)
	//if err != nil {
	//	return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	//}
	// check mandatory information in the input body
	err := licIn.CheckGetLicenseInput()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		return nil, http.Problem{Detail: err.Error(), Instance: licOut.ContentId, Status: http.StatusNotFound}
	} else if e != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// copy useful data from licIn to LicOut
	licIn.CopyInputToLicense(licOut)
	// build the license
	err = buildLicense(licOut, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// build a licenced publication
	publication, err := buildLicencedPublication(licOut, server)
	if err == filestor.ErrNotFound {
		return nil, http.Problem{Detail: err.Error(), Instance: licOut.ContentId, Status: http.StatusNotFound}
	} else if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := server.Store().Content().Get(licOut.ContentId)
	if err1 != nil {
		return nil, http.Problem{Detail: err1.Error(), Instance: licOut.ContentId, Status: http.StatusInternalServerError}
	}
	server.LogInfo("Content : %#v", content)
	// TODO : set below headers
	// -- set HTTP headers
	//resp.Header().Add(http.HdrContentType, epub.ContentTypeEpub)
	//resp.Header().Add(http.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	//resp.Header().Add(http.HdrXLcpLicense, licOut.Id)
	// -- must come *after* w.Header().Add()/Set(), but before w.Write()
	//resp.WriteHeader(http.StatusCreated)
	// -- return the full licensed publication to the caller
	// publication.Write(resp)
	return publication, nil
}

// GenerateLicensedPublication generates and returns a licensed publication
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicensedPublication(server http.IServer, lic *model.License, param ParamContentId) (*epub.Epub, error) {
	if param.ContentID == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}
	contentID := param.ContentID
	//server.LogInfo("Generate a Licensed publication for content id : %s", contentID)

	// get the input body
	//lic, err := ReadLicensePayload(req)
	//if err != nil {
	//	return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	//}
	// check mandatory information in the input body
	err := lic.CheckGenerateLicenseInput()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}

	}
	// init the license with an id and issue date
	err = lic.Initialize(contentID)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}
	// normalize the start and end date, UTC, no milliseconds
	lic.SetRights()
	// build the license
	err = buildLicense(lic, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	// store the license in the db
	err = server.Store().License().Add(lic)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Instance: contentID, Status: http.StatusInternalServerError}

	}

	// notify the lsd server of the creation of the license
	go notifyLSDServer(lic, server)

	// build a licenced publication
	publication, err := buildLicencedPublication(lic, server)
	if err == filestor.ErrNotFound {
		return nil, http.Problem{Detail: err.Error(), Instance: lic.ContentId, Status: http.StatusNotFound}

	} else if err != nil {
		return nil, http.Problem{Detail: err.Error(), Instance: lic.ContentId, Status: http.StatusInternalServerError}
	}

	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err1 := server.Store().Content().Get(lic.ContentId)
	if err1 != nil {
		return nil, http.Problem{Detail: err1.Error(), Instance: lic.ContentId, Status: http.StatusInternalServerError}
	}
	server.LogInfo("Content : %#v", content)
	// TODO : return headers
	// -- set HTTP headers
	//resp.Header().Add(http.HdrContentType, epub.ContentTypeEpub)
	//resp.Header().Add(http.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	//resp.Header().Add(http.HdrXLcpLicense, lic.Id)
	// -- must come *after* w.Header().Add()/Set(), but before w.Write()
	//resp.WriteHeader(http.StatusCreated)
	// -- return the full licensed publication to the caller
	//publication.Write(resp)
	return publication, nil
}

// UpdateLicense updates an existing license.
// parameters:
// 		{license_id} in the calling URL
// 		partial license containing properties which should be updated (and only these)
// return: an http status code (200, 400 or 404)
// Usually called from the License Status Server after a renew, return or cancel/revoke action
// -> updates the end date.
//
func UpdateLicense(server http.IServer, licIn *model.License, id ParamLicenseId) (*string, error) {
	if id.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}
	licenseID := id.LicenseID
	//licIn, err := ReadLicensePayload(req)
	//if err != nil { // no or incorrect (json) partial license found in the body
	//	server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
	//	return
	//}
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(licenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusNotFound}
	} else if e != nil {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusBadRequest}

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
	err := server.Store().License().Update(licOut)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	return nil, nil
}

// ListLicenses returns a JSON struct with information about the existing licenses
// parameters:
// 	page: page number
//	per_page: number of items par page
//
func ListLicenses(server http.IServer, id ParamContentIdAndPage) (model.LicensesCollection, error) {
	var err error
	page, _ := strconv.ParseInt(id.Page, 10, 64)
	perPage, _ := strconv.ParseInt(id.PerPage, 10, 64)
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 30
	}
	if page > 0 { //pagenum starting at 0 in code, but user interface starting at 1
		page--
	}
	if page < 0 {
		return nil, http.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest}
	}

	licenses, err := server.Store().License().ListAll(int(perPage), int(page))
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		server.LogInfo("Next page : %s", nextPage)
		//resp.Header().Set("LicenseLink", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
		//TODO : restore header
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		server.LogInfo("Next page : %s", previousPage)
		//TODO : restore header
		//resp.Header().Set("LicenseLink", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	//resp.Header().Set(http.HdrContentType, http.ContentTypeJson)

	return licenses, nil
}

// ListLicensesForContent lists all licenses associated with a given content
// parameters:
//	content_id: content identifier
// 	page: page number (default 1)
//	per_page: number of items par page (default 30)
//
func ListLicensesForContent(server http.IServer, param ParamContentIdAndPage) (model.LicensesCollection, error) {
	if param.ContentID == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}
	contentID := param.ContentID
	var err error
	//check if the license exists
	_, err = server.Store().Content().Get(contentID)
	if err == gorm.ErrRecordNotFound {
		server.LogInfo("License %s not found.", contentID)
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
	}

	page, _ := strconv.ParseInt(param.Page, 10, 64)
	perPage, _ := strconv.ParseInt(param.PerPage, 10, 64)
	//other errors pass, but will probably reoccur
	if page == 0 {
		page = 1
	}

	if perPage == 0 {
		perPage = 30
	}

	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		return nil, http.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest}

	}
	server.LogInfo("Listing licenses page %d with %d per page", page, perPage)
	licenses, err := server.Store().License().List(contentID, int(perPage), int(page))
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	if len(licenses) == 0 {
		return nil, http.Problem{Detail: "No licenses found for content with id " + param.ContentID, Status: http.StatusNotFound}
	}

	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		server.LogInfo("Next page : %s", nextPage)
		//TODO : restore header
		//resp.Header().Set("LicenseLink", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		server.LogInfo("Next page : %s", previousPage)
		//TODO : restore header
		//resp.Header().Set("LicenseLink", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	return licenses, nil

}

// ReadLicensePayload decodes a license formatted in json
//
func ReadLicensePayload(req *http.Request) (*model.License, error) {
	var dec *json.Decoder

	if ctype := req.Header[http.HdrContentType]; len(ctype) > 0 && ctype[0] == http.ContentTypeFormUrlEncoded {
		buf := bytes.NewBufferString(req.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(req.Body)
	}

	var result model.License
	return &result, dec.Decode(&result)
}
