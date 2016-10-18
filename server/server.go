// Copyright (c) 2016 Readium Founation
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

package server

import (
	"crypto/tls"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/server/api"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/technoweenie/grohl"

	"html/template"
	"net/http"
)

type Server struct {
	http.Server
	readonly bool
	idx      *index.Index
	st       *storage.Store
	lst      *license.Store
	router   *mux.Router
	cert     *tls.Certificate
	source   pack.ManualSource
}

func (s *Server) Store() storage.Store {
	return *s.st
}

func (s *Server) Index() index.Index {
	return *s.idx
}

func (s *Server) Licenses() license.Store {
	return *s.lst
}

func (s *Server) Certificate() *tls.Certificate {
	return s.cert
}

func (s *Server) Source() *pack.ManualSource {
	return &s.source
}

func New(bindAddr string, tplPath string, readonly bool, idx *index.Index, st *storage.Store, lst *license.Store, cert *tls.Certificate, packager *pack.Packager) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler: r,
			Addr:    bindAddr,
		},
		readonly: readonly,
		idx:      idx,
		st:       st,
		lst:      lst,
		cert:     cert,
		router:   r,
		source:   pack.ManualSource{},
	}

	s.source.Feed(packager.Incoming)

	manageIndex, err := template.ParseFiles(filepath.Join(tplPath, "/manage/index.html"))
	if err != nil {
		panic(err)
	}
	r.HandleFunc("/manage/", func(w http.ResponseWriter, r *http.Request) {
		manageIndex.Execute(w, map[string]interface{}{})
	})
	r.Handle("/manage/{file}", http.FileServer(http.Dir("static")))

	r.Handle("/files/{file}", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))
	if !readonly {
		s.handleFunc("/api/store/{name}", api.StorePackage).Methods("POST")
	}
	s.handleFunc("/api/packages", api.ListPackages).Methods("GET")
	s.handleFunc("/api/packages/{key}/licenses", api.GrantLicense).Methods("POST")
	r.Handle("/", http.NotFoundHandler())

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s api.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		grohl.Log(grohl.Data{"path": r.URL.Path})

		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
