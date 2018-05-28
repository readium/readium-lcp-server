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
	"strconv"

	"fmt"
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
func GetLicense(server http.IServer, payload *model.License, param ParamLicenseId) (*model.License, error) {
	if param.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}

	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(param.LicenseID)
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
		payload, err := ReadLicensePayload(req)
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
	err := payload.ValidateEncryption()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// copy useful data from payload to LicOut
	payload.CopyInputToLicense(licOut)
	// build the license
	err = BuildLicense(licOut, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	// TODO : set below headers
	// set the http headers
	nonErr.HttpHeaders.Add(http.HdrContentType, http.ContentTypeLcpJson)
	nonErr.HttpHeaders.Add(http.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	//resp.WriteHeader(http.StatusOK)
	// send back the license
	// TODO : do not escape characters in the json payload
	return licOut, nonErr
}

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(server http.IServer, payload *model.License, param ParamContentId) (*model.License, error) {
	if param.ContentID == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}

	panic("Test")
	// check mandatory information in the input body
	err := payload.ValidateProviderAndUser()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	//server.LogInfo("Initialize.")
	// init the license with an id and issue date
	err = payload.Initialize(param.ContentID)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	//server.LogInfo("Setting rights.")
	// normalize the start and end date, UTC, no milliseconds
	payload.SetRights()

	//server.LogInfo("Building license.")
	// build the license
	err = BuildLicense(payload, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	//jsonPayload, err := json.MarshalIndent(payload, " ", " ")
	//server.LogInfo("Saving to database : %s", string(jsonPayload))
	// store the license in the db
	err = server.Store().License().Add(payload)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	nonErr := http.Problem{Status: http.StatusCreated, HttpHeaders: make(map[string][]string)}
	// TODO : set below headers
	// set http headers
	nonErr.HttpHeaders.Add(http.HdrContentType, http.ContentTypeLcpJson)
	nonErr.HttpHeaders.Add(http.HdrContentDisposition, `attachment; filename="license.lcpl"`)
	// TODO : do not escape characters
	//	send back the license
	//enc := json.NewEncoder(resp)
	//enc.SetEscapeHTML(false)
	//enc.Encode(payload)

	// notify the lsd server of the creation of the license.
	// this is an asynchronous call.
	go notifyLSDServer(payload, server)
	return payload, nonErr
}

// GetLicensedPublication returns a licensed publication
// for a given license identified by its id
// plus a partial license given as input
//
func GetLicensedPublication(server http.IServer, payload *model.License, param ParamLicenseId) (*epub.Epub, error) {
	if param.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}
	// get the input body
	//payload, err := ReadLicensePayload(req)
	//if err != nil {
	//	return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	//}
	// check mandatory information in the input body
	err := payload.ValidateEncryption()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// initialize the license from the info stored in the db.
	licOut, e := server.Store().License().Get(param.LicenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		return nil, http.Problem{Detail: err.Error(), Instance: licOut.ContentId, Status: http.StatusNotFound}
	} else if e != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// copy useful data from payload to LicOut
	payload.CopyInputToLicense(licOut)
	// build the license
	err = BuildLicense(licOut, server)
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
	nonErr := http.Problem{Status: http.StatusCreated, HttpHeaders: make(map[string][]string)}
	// TODO : set below headers
	// -- set HTTP headers
	nonErr.HttpHeaders.Add(http.HdrContentType, epub.ContentTypeEpub)
	nonErr.HttpHeaders.Add(http.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	nonErr.HttpHeaders.Add(http.HdrXLcpLicense, licOut.Id)
	// -- must come *after* w.Header().Add()/Set(), but before w.Write()

	// -- return the full licensed publication to the caller
	// publication.Write(resp)
	return publication, nonErr
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
	// check mandatory information in the input body
	err := lic.ValidateProviderAndUser()
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
	err = BuildLicense(lic, server)
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
	nonErr := http.Problem{Status: http.StatusCreated, HttpHeaders: make(map[string][]string)}
	server.LogInfo("Content : %#v", content)
	// TODO : return headers
	// -- set HTTP headers
	nonErr.HttpHeaders.Add(http.HdrContentType, epub.ContentTypeEpub)
	nonErr.HttpHeaders.Add(http.HdrContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	// FIXME: check the use of X-Lcp-License by the caller (frontend?)
	nonErr.HttpHeaders.Add(http.HdrXLcpLicense, lic.Id)
	// -- must come *after* w.Header().Add()/Set(), but before w.Write()

	// -- return the full licensed publication to the caller
	//publication.Write(resp)
	return publication, nonErr
}

// UpdateLicense updates an existing license.
// parameters:
// 		{license_id} in the calling URL
// 		partial license containing properties which should be updated (and only these)
// return: an http status code (200, 400 or 404)
// Usually called from the License Status Server after a renew, return or cancel/revoke action
// -> updates the end date.
//
func UpdateLicense(server http.IServer, payload *model.License, param ParamLicenseId) (*string, error) {
	if param.LicenseID == "" {
		return nil, http.Problem{Detail: "The license id must be set in the url", Status: http.StatusBadRequest}
	}
	// initialize the license from the info stored in the db.
	existingLicense, e := server.Store().License().Get(param.LicenseID)
	// process license not found etc.
	if e == gorm.ErrRecordNotFound {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusNotFound}
	} else if e != nil {
		return nil, http.Problem{Detail: e.Error(), Status: http.StatusBadRequest}
	}
	existingLicense.Update(payload)
	// update the license in the database
	err := server.Store().License().Update(existingLicense)
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
func ListLicenses(server http.IServer, param http.ParamPagination) (model.LicensesCollection, error) {
	noOfLicenses, err := server.Store().License().Count()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	if noOfLicenses == 0 {
		return nil, http.Problem{Detail: "no licenses found", Status: http.StatusNotFound}
	}

	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfLicenses)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	licenses, err := server.Store().License().ListAll(perPage, page)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Set("Link", http.MakePaginationHeader("http://localhost:"+strconv.Itoa(server.Config().LcpServer.Port)+"/licenses", page+1, perPage, noOfLicenses))
	return licenses, nonErr
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
	noOfLicenses, err := server.Store().License().CountForContentId(contentID)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	if noOfLicenses == 0 {
		return nil, http.Problem{Detail: "No licenses found for content with id " + param.ContentID, Status: http.StatusNotFound}
	}

	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfLicenses)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	licenses, err := server.Store().License().List(contentID, perPage, page)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Set("Link", http.MakePaginationHeader("http://localhost:"+strconv.Itoa(server.Config().LcpServer.Port)+"/licenses", page+1, perPage, noOfLicenses))
	return licenses, nonErr

}
