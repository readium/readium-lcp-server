// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package problem

// rfc 7807 : https://tools.ietf.org/html/rfc7807
// problem.Type should be an URI
// for example http://readium.org/readium/[lcpserver|lsdserver]/<code>
// for standard http error messages use "about:blank" status in json equals http status
import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/readium/readium-lcp-server/logging"
)

const (
	ContentType_PROBLEM_JSON = "application/problem+json"
)

type Problem struct {
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
	//optional
	Status   int    `json:"status,omitempty"` //if present = http response code
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// Problem types
const ERROR_BASE_URL = "http://readium.org/license-status-document/error/"
const LICENSE_NOT_FOUND = ERROR_BASE_URL + "notfound"
const SERVER_INTERNAL_ERROR = ERROR_BASE_URL + "server"
const REGISTRATION_BAD_REQUEST = ERROR_BASE_URL + "registration"
const RETURN_BAD_REQUEST = ERROR_BASE_URL + "return"
const RETURN_EXPIRED = ERROR_BASE_URL + "return/expired"
const RETURN_ALREADY = ERROR_BASE_URL + "return/already"
const RENEW_BAD_REQUEST = ERROR_BASE_URL + "renew"
const RENEW_REJECT = ERROR_BASE_URL + "renew/date"
const CANCEL_BAD_REQUEST = ERROR_BASE_URL + "cancel"
const FILTER_BAD_REQUEST = ERROR_BASE_URL + "filter"

func Error(w http.ResponseWriter, r *http.Request, problem Problem, status int) {

	w.Header().Set("Content-Type", ContentType_PROBLEM_JSON)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(status)

	problem.Status = status

	if problem.Type == "" && status == http.StatusInternalServerError {
		problem.Type = SERVER_INTERNAL_ERROR
	}

	if problem.Title == "" { // Title (required) matches http status by default
		problem.Title = http.StatusText(status)
	}

	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	fmt.Fprintln(w, string(jsonError))

	// debug only
	//PrintStack()

	msg := fmt.Sprintf("Error: %s (%d) = %s", problem.Title, problem.Status, problem.Detail)
	// log the error persistently
	logging.Print(msg)
}

func PrintStack() {
	log.Print("####################")

	//debug.PrintStack()

	b := debug.Stack()
	s := string(b[:])
	l := strings.Index(s, "ServeHTTP")
	if l > 0 {
		ss := s[0:l]
		log.Print(ss + " [...]")
	} else {
		log.Print(s)
	}

	log.Print("####################")
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	var problem Problem
	problem.Type = LICENSE_NOT_FOUND
	problem.Title = "Failed to find the license ID"
	Error(w, r, problem, http.StatusNotFound)
}
