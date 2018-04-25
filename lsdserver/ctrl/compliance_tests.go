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

package apilsd

import (
	"net/http"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/logging"
)

// Possible values of test stage
const (
	LicenseStatus       = "status"
	RegistDevice        = "register"
	RenewLicense        = "renew"
	ReturnLicense       = "return"
	CancelRevokeLicense = "revoke"
)

var (
	complianceTestNumber string

	results = map[string]string{
		"s": "success",
		"e": "error",
	}
)

// AddLogToFile adds a log message to the log file
//
func AddLogToFile(resp http.ResponseWriter, req *http.Request, server IServer) {
	testStage := req.FormValue("test_stage")
	testNumber := req.FormValue("test_number")
	testResult := req.FormValue("test_result")

	server.LogInfo("compliance test number %v, %v, result %v", testNumber, testStage, testResult)

	if testStage != "start" && testStage != "end" {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Type: "about:blank", Detail: "The stage of the compliance test must be either 'start' or 'end'"}, http.StatusBadRequest)
		return
	}

	if testStage == "start" {
		if len(testNumber) == 0 {
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Type: "about:blank", Detail: "The number of compliance test cannot be null"}, http.StatusBadRequest)
		} else {
			complianceTestNumber = testNumber
			testResult = "-"
			logging.WriteToFile(complianceTestNumber, testStage, testResult, "")
		}
	} else {
		if testResult != "e" && testResult != "s" {
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Type: "about:blank", Detail: "The result of compliance test must be either 'e' or 's'"}, http.StatusBadRequest)
		} else {
			testResult = results[testResult]
			logging.WriteToFile(complianceTestNumber, testStage, testResult, "")
			complianceTestNumber = ""
		}
	}
}
