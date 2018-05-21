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

// rfc 7807 : https://tools.ietf.org/html/rfc7807
// problem.Type should be an URI
// for example http://readium.org/readium/[lcpserver|lsdserver]/<code>
// for standard http error messages use "about:blank" status in json equals http status
import (
	"encoding/json"
	"fmt"
	"github.com/technoweenie/grohl"
	"log"
	"net/http"

	"bytes"
	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/jeffbmartinez/delay"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/store"
	"github.com/rs/cors"
	"github.com/urfave/negroni"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func CreateServerRouter(tplPath string) ServerRouter {

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(NotFoundHandler) //handle all other requests 404

	// this demonstrates a panic report
	r.HandleFunc("/panic", func(w http.ResponseWriter, req *http.Request) {
		panic("just testing. no worries.")
	})

	//n := negroni.Classic() == negroni.New(negroni.NewRecovery(), negroni.NewLogger(), negroni.NewStatic(...))
	n := negroni.New()

	// HTTP client can emit requests with custom header:
	//X-Add-Delay: 300ms
	//X-Add-Delay: 2.5s
	n.Use(delay.Middleware{})

	// possibly useful middlewares:
	// https://github.com/jeffbmartinez/delay

	//https://github.com/urfave/negroni#recovery
	recovery := negroni.NewRecovery()
	recovery.PrintStack = true
	recovery.ErrorHandlerFunc = PanicReport
	n.Use(recovery)

	//https://github.com/urfave/negroni#logger
	//n.Use(negroni.NewLogger())

	// debug: log request details
	//n.Use(negroni.HandlerFunc(ExtraLogger))

	if tplPath != "" {
		//https://github.com/urfave/negroni#static
		n.Use(negroni.NewStatic(http.Dir(tplPath)))
	}

	// debug: log CORS details
	//n.Use(negroni.HandlerFunc(CORSHeaders))

	// Does not insert CORS headers as intended, depends on Origin check in the HTTP request...we want the same headers, always.
	// IMPORT "github.com/rs/cors"
	// //https://github.com/rs/cors#parameters
	// [cors] logs depend on the Debug option (false/true)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"PATCH", "HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders: []string{"Range", "Content-Type", "Origin", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Authorization"},
		Debug:          false,
	})
	n.Use(c)

	n.UseHandler(r)

	sr := ServerRouter{
		R: r,
		N: n,
	}

	return sr
}

func ExtraLogger(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	log.Print(" << -------------------")

	fmt.Printf("%s => %s (%s)\n", r.RemoteAddr, r.URL.String(), r.RequestURI)

	grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery})

	log.Printf("REQUEST headers: %#v", r.Header)

	// before
	next(rw, r)
	// after

	contentType := rw.Header().Get(HdrContentType)
	if contentType == ContentTypeProblemJson {
		log.Print("^^^^ " + ContentTypeProblemJson + " ^^^^")
	}

	log.Printf("RESPONSE headers: %#v", rw.Header())

	log.Print(" >> -------------------")
}

func CORSHeaders(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	grohl.Log(grohl.Data{"CORS": "yes"})
	rw.Header().Add("Access-Control-Allow-Methods", "PATCH, HEAD, POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Add("Access-Control-Allow-Credentials", "true")
	rw.Header().Add("Access-Control-Allow-Origin", "*")
	rw.Header().Add("Access-Control-Allow-Headers", "Range, Content-Type, Origin, X-Requested-With, Accept, Accept-Language, Content-Language, Authorization")

	// before
	next(rw, r)
	// after

	// noop
}

func CheckAuth(authenticator *auth.BasicAuth, w http.ResponseWriter, r *http.Request) bool {
	var username string
	if username = authenticator.CheckAuth(r); username == "" {
		grohl.Log(grohl.Data{"error": "Unauthorized", "method": r.Method, "path": r.URL.Path})
		w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
		Error(w, r, "en_US", Problem{Detail: "User or password do not match!"}, http.StatusUnauthorized)
		return false
	}
	grohl.Log(grohl.Data{"user": username})
	return true
}

func Error(w http.ResponseWriter, r *http.Request, defautServerLang string, problem Problem, status int) {
	acceptLanguages := r.Header.Get("Accept-Language")

	w.Header().Set(HdrContentType, ContentTypeProblemJson)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(status)

	problem.Status = status

	if problem.Type == "about:blank" || problem.Type == "" { // lookup Title  statusText should match http status
		localization.LocalizeMessage(defautServerLang, acceptLanguages, &problem.Title, http.StatusText(status))
	} else {
		localization.LocalizeMessage(defautServerLang, acceptLanguages, &problem.Title, problem.Title)
		localization.LocalizeMessage(defautServerLang, acceptLanguages, &problem.Detail, problem.Detail)
	}
	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	fmt.Fprintln(w, string(jsonError))

	// debug only
	//PrintStack()

	log.Print(string(jsonError))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path, "status": "404"})
	Error(w, r, "en_US", Problem{}, http.StatusNotFound)
}

func PanicReport(err interface{}) {
	switch t := err.(type) {
	case error:
		errorr, found := err.(error)
		if found { // should always be true
			grohl.Log(grohl.Data{"panic recovery (error)": errorr.Error()})
		}
	case string:
		errorr, found := err.(string)
		if found { // should always be true
			grohl.Log(grohl.Data{"panic recovery (string)": errorr})
		}
	default:
		grohl.Log(grohl.Data{"panic recovery (other type)": t})
	}
}

//ReadUserPayload transforms a json string to a User struct
func ReadUserPayload(req *http.Request) (*store.User, error) {
	var dec *json.Decoder
	if ctype := req.Header[HdrContentType]; len(ctype) > 0 && ctype[0] == ContentTypeJson {
		dec = json.NewDecoder(req.Body)
	}
	user := &store.User{}
	err := dec.Decode(user)
	return user, err
}

//ReadPublicationPayload transforms a json
func ReadPublicationPayload(req *http.Request) (*store.Publication, error) {
	var dec *json.Decoder
	if ctype := req.Header[HdrContentType]; len(ctype) > 0 && ctype[0] == ContentTypeJson {
		dec = json.NewDecoder(req.Body)
	}

	var result store.Publication
	return &result, dec.Decode(&result)
}

// ReadPurchasePayload transforms a json
//
func ReadPurchasePayload(req *http.Request) (*store.Purchase, error) {
	var dec *json.Decoder
	if ctype := req.Header[HdrContentType]; len(ctype) > 0 && ctype[0] == ContentTypeJson {
		dec = json.NewDecoder(req.Body)
	}

	var result store.Purchase
	return &result, dec.Decode(&result)
}

// ReadLicensePayload decodes a license formatted in json
//
func ReadLicensePayload(req *http.Request) (*store.License, error) {
	var dec *json.Decoder

	if ctype := req.Header[HdrContentType]; len(ctype) > 0 && ctype[0] == ContentTypeFormUrlEncoded {
		buf := bytes.NewBufferString(req.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(req.Body)
	}

	var result store.License
	return &result, dec.Decode(&result)
}

// ReadLicenseStatusPayload decodes license status json to the object
//
func ReadLicenseStatusPayload(r *http.Request) (*store.LicenseStatus, error) {
	var dec *json.Decoder

	if ctype := r.Header[HdrContentType]; len(ctype) > 0 && ctype[0] == ContentTypeFormUrlEncoded {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	var result store.LicenseStatus
	return &result, dec.Decode(&result)
}

func ReadLicensesPayloads(data []byte) (store.LicensesStatusCollection, error) {
	var licenses store.LicensesStatusCollection
	err := json.Unmarshal(data, &licenses)
	if err != nil {
		return nil, err
	}
	return licenses, nil
}

func HandleSignals() {
	sigChan := make(chan os.Signal)
	go func() {
		stacktrace := make([]byte, 1<<20)
		for sig := range sigChan {
			switch sig {
			case syscall.SIGQUIT:
				length := runtime.Stack(stacktrace, true)
				fmt.Println(string(stacktrace[:length]))
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				fmt.Println("Shutting down...")
				os.Exit(0)
			}
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
}
