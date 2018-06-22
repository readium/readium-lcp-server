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
	"bytes"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/views"
)

// GetFilteredLicenses searches licenses activated by more than n devices
//
func GetFilteredLicenses(server http.IServer, param ParamPagination) (*views.Renderer, error) {
	view := &views.Renderer{}
	if param.Filter == "" {
		param.Filter = "1"
	} else {
		view.AddKey("filter", param.Filter)
	}
	noOfLicenses, err := server.Store().License().CountFiltered(param.Filter)
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfLicenses)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	licenses, err := server.Store().License().GetFiltered(param.Filter, perPage, page)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	view.AddKey("licenses", licenses)
	view.AddKey("pageTitle", "Licenses list")
	view.AddKey("total", noOfLicenses)
	view.AddKey("currentPage", 1)
	view.AddKey("perPage", 20)
	view.Template("licenses/index.html.got")
	return view, nil
}

// GetLicense gets an existing license by its id (passed as a section of the REST URL).
// It generates a partial license from the purchase info,
// fetches the license from the lcp server and returns it to the caller.
// This API method is called from a link in the license status document.
//
func GetLicense(server http.IServer, param ParamId) ([]byte, error) {
	purchase, err := server.Store().Purchase().GetByLicenseID(param.Id)
	// get the license id in the URL
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	// get an existing license from the lcp server
	fullLicense, err := generateOrGetLicense(purchase, server)
	if err != nil {
		server.LogError("Error generating or getting license : %v", err)
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	// message to the console
	//server.LogInfo("Get license / id " + param.Id + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
	//server.LogInfo("Full license : %#v", fullLicense)
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Add(http.HdrContentType, http.ContentTypeLcpJson)
	nonErr.HttpHeaders.Set(http.HdrContentDisposition, "attachment; filename=\"license.lcpl\"")
	result := new(bytes.Buffer)
	enc := json.NewEncoder(result)
	// do not escape characters
	enc.SetEscapeHTML(false)
	// send back the license
	enc.Encode(fullLicense)

	return result.Bytes(), nonErr
}
