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

package http

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/i18n"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
)

func (s *Server) GoofyMode() bool {
	return s.GoophyMode
}

// CreateDefaultLinks inits the global var DefaultLinks from config data
// ... DefaultLinks used in several places.
//
func (s *Server) CreateDefaultLinks(cfg License) {
	s.DefaultLinks = make(map[string]string)
	for key := range cfg.Links {
		s.DefaultLinks[key] = cfg.Links[key]
	}
}

// SetDefaultLinks sets a LicenseLink array from config links
//
func (s *Server) SetDefaultLinks() model.LicenseLinksCollection {
	links := make(model.LicenseLinksCollection, 0, 0)
	for key := range s.DefaultLinks {
		links = append(links, &model.LicenseLink{Href: s.DefaultLinks[key], Rel: key})
	}
	return links
}

// SetLicenseLinks sets publication and status links
// l.ContentId must have been set before the call
//
func (s *Server) SetLicenseLinks(l *model.License, c *model.Content) error {
	// set the links
	l.Links = s.SetDefaultLinks()

	for i := 0; i < len(l.Links); i++ {
		switch l.Links[i].Rel {
		// publication link
		case "publication":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{publication_id}", l.ContentId, 1)
			l.Links[i].Type = epub.ContentTypeEpub
			l.Links[i].Size = c.Length
			l.Links[i].Title = c.Location
			l.Links[i].Checksum = c.Sha256
			// status link
		case "status":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{license_id}", l.Id, 1)
			l.Links[i].Type = ContentTypeLsdJson

		}

	}
	return nil
}

func (s *Server) Storage() filestor.Store {
	return *s.St
}

func (s *Server) Store() model.Store {
	return s.Model
}

func (s *Server) Certificate() *tls.Certificate {
	return s.Cert
}

func (s *Server) Source() *pack.ManualSource {
	return &s.Src
}

func (s *Server) Config() Configuration {
	return s.Cfg
}

func (s *Server) DefaultSrvLang() string {
	if s.Cfg.Localization.DefaultLanguage == "" {
		return "en_US"
	}
	return s.Cfg.Localization.DefaultLanguage
}

func (s *Server) LogError(format string, args ...interface{}) {
	s.Log.Errorf(format, args...)
}

func (s *Server) LogInfo(format string, args ...interface{}) {
	s.Log.Infof(format, args...)
}

func (s *Server) HandleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, s)
	})
}

func (s *Server) Error(w http.ResponseWriter, r *http.Request, problem Problem) {
	acceptLanguages := r.Header.Get("Accept-Language")

	w.Header().Set(HdrContentType, ContentTypeProblemJson)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(problem.Status)

	if problem.Type == "about:blank" || problem.Type == "" { // lookup Title  statusText should match http status
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, http.StatusText(problem.Status))
	} else {
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, problem.Title)
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Detail, problem.Detail)
	}
	jsonError, e := json.Marshal(problem)
	if e != nil {
		s.Log.Errorf("Error serializing problem : %v", e)
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonError)

	s.Log.Infof("Handled error : %s", string(jsonError))
}

func (s *Server) HandlePrivateFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if username := s.checkAuth(r); username == "" {
			s.Log.Errorf("method=%s path=%s error=Unauthorized", r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Basic realm="`+s.realm+`"`)
			s.Error(w, r, Problem{Detail: "User or password do not match!", Status: http.StatusUnauthorized})
		} else {
			s.Log.Infof("user=%s", username)
			fn(w, r, s)
		}
	})
}

func (s *Server) MakeAutorizator(realm string) {
	authFile := s.Cfg.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}
	s.secretProvider = HtpasswdFileProvider(authFile)
}

func (s *Server) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Log.Infof("method=%s path=%s status=404", r.Method, r.URL.Path)
		s.Error(w, r, Problem{Status: http.StatusNotFound})
	}
}
