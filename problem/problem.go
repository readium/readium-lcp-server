// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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

// NotFoundHandler handles 404 API errors
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	Error(w, r, Problem{Detail: r.URL.String()}, http.StatusNotFound)
}
