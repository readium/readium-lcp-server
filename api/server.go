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
	"context"
	"crypto/tls"
	"encoding/base64"
	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/readium/readium-lcp-server/store"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
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
func (s *Server) SetDefaultLinks() store.LicenseLinksCollection {
	links := make(store.LicenseLinksCollection, 0, 0)
	for key := range s.DefaultLinks {
		links = append(links, &store.LicenseLink{Href: s.DefaultLinks[key], Rel: key})
	}
	return links
}

// SetLicenseLinks sets publication and status links
// l.ContentId must have been set before the call
//
func (s *Server) SetLicenseLinks(l *store.License, c *store.Content) error {
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

func (s *Server) Storage() storage.Store {
	return *s.St
}

func (s *Server) Store() store.Store {
	return s.ORM
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

func (s *Server) HandlePrivateFunc(router *mux.Router, route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if CheckAuth(authenticator, w, r) {
			fn(w, r, s)
		}
	})
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
	err = s.ORM.License().PurgeDataBase()
	if err != nil {
		panic(err)
	}

	licenses, err := ReadLicensesPayloads(body)
	if err != nil {
		panic(err)
	}
	// fill the db
	err = s.ORM.License().BulkAdd(licenses)
	if err != nil {
		panic(err)
	}
}
