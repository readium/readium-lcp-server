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

package lutserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/controller/common"
)

// GetFilteredLicenses searches licenses activated by more than n devices
//
func GetFilteredLicenses(resp http.ResponseWriter, req *http.Request, server common.IServer) {

	rDevices := req.FormValue("devices")
	log.Println("Licenses used by " + rDevices + " devices")
	if rDevices == "" {
		rDevices = "0"
	}

	if lic, err := server.Store().License().GetFiltered(rDevices); err == nil {
		resp.Header().Set(common.HdrContentType, common.ContentTypeJson)
		enc := json.NewEncoder(resp)
		if err = enc.Encode(lic); err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		}
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			{
				server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusNotFound})
			}
		default:
			{
				server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			}
		}
	}
}

// GetLicense gets an existing license by its id (passed as a section of the REST URL).
// It generates a partial license from the purchase info,
// fetches the license from the lcp server and returns it to the caller.
// This API method is called from a link in the license status document.
//
func GetLicense(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)
	var licenseID = vars["license_id"]
	purchase, err := server.Store().Purchase().GetByLicenseID(licenseID)
	// get the license id in the URL
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusNotFound})
		default:
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		}
		return
	}
	// get an existing license from the lcp server
	fullLicense, err := generateOrGetLicense(purchase, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	// return a json payload
	resp.Header().Set(common.HdrContentType, common.ContentTypeLcpJson)
	// the file name is license.lcpl
	resp.Header().Set(common.HdrContentDisposition, "attachment; filename=\"license.lcpl\"")

	// returns the full license to the caller
	enc := json.NewEncoder(resp)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(fullLicense)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
	// message to the console
	log.Println("Get license / id " + licenseID + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
}
