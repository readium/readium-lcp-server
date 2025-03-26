// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package api

import (
	"fmt"
	"log"
	"net/http"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/jeffbmartinez/delay"
	"github.com/rs/cors"
	"github.com/urfave/negroni"

	"github.com/readium/readium-lcp-server/problem"
)

const (
	// DO NOT FORGET to update the version
	Software_Version = "1.10.0"

	ContentType_LCP_JSON  = "application/vnd.readium.lcp.license.v1.0+json"
	ContentType_LSD_JSON  = "application/vnd.readium.license.status.v1.0+json"
	ContentType_TEXT_HTML = "text/html"

	ContentType_JSON = "application/json"

	ContentType_FORM_URL_ENCODED = "application/x-www-form-urlencoded"
)

type ServerRouter struct {
	R *mux.Router
	N *negroni.Negroni
}

func CreateServerRouter(tplPath string) ServerRouter {

	log.Println("Software Version " + Software_Version)

	r := mux.NewRouter()

	//handle 404 errors
	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler)

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
	n.Use(recovery)

	//https://github.com/urfave/negroni#logger
	// Nov 2023, suppression of negroni logs
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

	log.Println("method: ", r.Method, " path: ", r.URL.Path, " query: ", r.URL.RawQuery)

	log.Printf("REQUEST headers: %#v", r.Header)

	// before
	next(rw, r)
	// after

	contentType := rw.Header().Get("Content-Type")
	if contentType == problem.ContentType_PROBLEM_JSON {
		log.Print("^^^^ " + problem.ContentType_PROBLEM_JSON + " ^^^^")
	}

	log.Printf("RESPONSE headers: %#v", rw.Header())

	log.Print(" >> -------------------")
}

func CORSHeaders(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

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
		w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
		problem.Error(w, r, problem.Problem{Detail: "User or password do not match!"}, http.StatusUnauthorized)
		return false
	}
	return true
}
