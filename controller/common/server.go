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

package common

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/file_storage"
	"github.com/readium/readium-lcp-server/lib/localization"
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

func (s *Server) Storage() file_storage.Store {
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
		localization.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, http.StatusText(problem.Status))
	} else {
		localization.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, problem.Title)
		localization.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Detail, problem.Detail)
	}
	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	//fmt.Fprintln(w, string(jsonError))

	// debug only
	//PrintStack()

	s.Log.Infof(string(jsonError))
}

func (s *Server) HandlePrivateFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if username := s.authenticator.CheckAuth(r); username == "" {
			s.Log.Errorf("method=%s path=%s error=Unauthorized", r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Basic realm="`+s.authenticator.Realm+`"`)
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
	htpasswd := HtpasswdFileProvider(authFile)
	s.authenticator = NewBasicAuthenticator(realm, htpasswd)
}

func (s *Server) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Log.Infof("method=%s path=%s status=404", r.Method, r.URL.Path)
		s.Error(w, r, Problem{Status: http.StatusNotFound})
	}
}

func (s *Server) FetchLicenseStatusesFromLSD() {
	s.Log.Printf("AUTOMATION : Fetch and save all license status documents")

	url := s.Config().LsdServer.PublicBaseUrl + "/licenses"
	auth := s.Config().LsdNotifyAuth

	// prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth.Username+":"+auth.Password)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		s.Log.Printf("AUTOMATION : Error getting license statuses : %v", err)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.Log.Fatalf("AUTOMATION : Error reading response body error : %v", err)
	}

	s.Log.Printf("AUTOMATION : lsd server response : %v [http-status:%d]", body, resp.StatusCode)

	// clear the db
	err = s.Model.License().PurgeDataBase()
	if err != nil {
		panic(err)
	}

	licenses, err := ReadLicensesPayloads(body)
	if err != nil {
		panic(err)
	}
	// fill the db
	err = s.Model.License().BulkAdd(licenses)
	if err != nil {
		panic(err)
	}
}
