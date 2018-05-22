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

	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
)

// GetDashboardInfos searches a publication by its uuid
func GetDashboardInfos(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	if pub, err := server.Store().Dashboard().GetDashboardInfos(); err == nil {
		enc := json.NewEncoder(resp)
		if err = enc.Encode(pub); err == nil {
			// send json of correctly encoded user info
			resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
			resp.WriteHeader(http.StatusOK)
			return
		}

		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusNotFound})
			}
		default:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			}
		}
	}
}

// GetDashboardBestSellers gets the dashboard bestsellers
//
func GetDashboardBestSellers(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	if pub, err := server.Store().Dashboard().GetDashboardBestSellers(); err == nil {
		enc := json.NewEncoder(resp)
		if err = enc.Encode(pub); err == nil {
			// send json of correctly encoded user info
			resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
			resp.WriteHeader(http.StatusOK)
			return
		}

		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusNotFound})
			}
		default:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			}
		}
	}
}
