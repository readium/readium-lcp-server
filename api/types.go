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

package api

import (
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

const (
	ErrorBaseUrl = "http://readium.org/license-status-document/error/"

	ServerInternalError    = ErrorBaseUrl + "server"
	RegistrationBadRequest = ErrorBaseUrl + "registration"
	ReturnBadRequest       = ErrorBaseUrl + "return"
	RenewBadRequest        = ErrorBaseUrl + "renew"
	RenewReject            = ErrorBaseUrl + "renew/date"
	CancelBadRequest       = ErrorBaseUrl + "cancel"
	FilterBadRequest       = ErrorBaseUrl + "filter"

	HdrContentType            = "Content-Type"
	HdrContentDisposition     = "Content-Disposition"
	HdrXLcpLicense            = "X-Lcp-License"
	ContentTypeProblemJson    = "application/problem+json"
	ContentTypeLcpJson        = "application/vnd.readium.lcp.license.v1.0+json"
	ContentTypeLsdJson        = "application/vnd.readium.license.status.v1.0+json"
	ContentTypeJson           = "application/json"
	ContentTypeFormUrlEncoded = "application/x-www-form-urlencoded"
)

type (
	ServerRouter struct {
		R *mux.Router
		N *negroni.Negroni
	}

	// LcpPublication is a struct for communication with lcp-server
	LcpPublication struct {
		ContentId          string  `json:"content-id"`
		ContentKey         []byte  `json:"content-encryption-key"`
		Output             string  `json:"protected-content-location"`
		Size               *int64  `json:"protected-content-length,omitempty"`
		Checksum           *string `json:"protected-content-sha256,omitempty"`
		ContentDisposition *string `json:"protected-content-disposition,omitempty"`
		ErrorMessage       string  `json:"error"`
	}

	Problem struct {
		Status   int    `json:"status,omitempty"` // if present = http response code
		Type     string `json:"type,omitempty"`
		Title    string `json:"title,omitempty"`
		Detail   string `json:"detail,omitempty"`
		Instance string `json:"instance,omitempty"`
	}
)
