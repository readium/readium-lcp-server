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

package lcpserver

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/epub"
	LCPCtrl "github.com/readium/readium-lcp-server/lcpserver/ctrl"
	"github.com/readium/readium-lcp-server/logger"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/readium/readium-lcp-server/store"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type (
	Server struct {
		http.Server
		readonly     bool
		log          logger.StdLogger
		store        store.Store
		st           *storage.Store
		cert         *tls.Certificate
		source       pack.ManualSource
		config       api.Configuration
		DefaultLinks map[string]string
	}

	HandlerFunc func(w http.ResponseWriter, r *http.Request, s LCPCtrl.IServer)

	HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s LCPCtrl.IServer)
)

// CreateDefaultLinks inits the global var DefaultLinks from config data
// ... DefaultLinks used in several places.
//
func (s *Server) CreateDefaultLinks(cfg api.License) {
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
			l.Links[i].Type = api.ContentTypeLsdJson

		}

	}
	return nil
}

func (s *Server) Storage() storage.Store {
	return *s.st
}

func (s *Server) Store() store.Store {
	return s.store
}

func (s *Server) Certificate() *tls.Certificate {
	return s.cert
}

func (s *Server) Source() *pack.ManualSource {
	return &s.source
}

func (s *Server) Config() api.Configuration {
	return s.config
}

func (s *Server) DefaultSrvLang() string {
	return s.config.Localization.DefaultLanguage
}

func (s *Server) LogError(format string, args ...interface{}) {
	s.log.Errorf(format, args...)
}

func (s *Server) LogInfo(format string, args ...interface{}) {
	s.log.Infof(format, args...)
}

func (s *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, s)
	})
}

func (s *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if api.CheckAuth(authenticator, w, r) {
			fn(w, r, s)
		}
	})
}

func New(
	cfg api.Configuration,
	log logger.StdLogger,
	stor store.Store,
	storage *storage.Store,
	cert *tls.Certificate,
	packager *pack.Packager,
	basicAuth *auth.BasicAuth) *Server {

	parsedPort := strconv.Itoa(cfg.LcpServer.Port)

	readonly := cfg.LcpServer.ReadOnly

	// writing static
	static := cfg.LcpServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../web/lcp")
	}
	filepathConfigJs := filepath.Join(static, "/config.js")
	fileConfigJs, err := os.Create(filepathConfigJs)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fileConfigJs.Close(); err != nil {
			panic(err)
		}
	}()

	static = cfg.LcpServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../lcpserver/manage")
	}
	configJs := "// This file is automatically generated, and git-ignored.\n// To ignore your local changes, use:\n// git update-index --assume-unchanged lcpserver/manage/config.js\n\nvar Config = {\n    lcp: {url: '" + cfg.LcpServer.PublicBaseUrl + "', user:'" + cfg.LcpUpdateAuth.Username + "', password: '" + cfg.LcpUpdateAuth.Password + "'},\n    lsd: {url: '" + cfg.LsdServer.PublicBaseUrl + "', user:'" + cfg.LcpUpdateAuth.Username + "', password: '" + cfg.LcpUpdateAuth.Password + "'}\n}\n"

	log.Printf("manage/index.html config.js:")
	log.Printf(configJs)
	fileConfigJs.WriteString(configJs)

	sr := api.CreateServerRouter(static)

	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           ":" + parsedPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		log:      log,
		config:   cfg,
		readonly: readonly,
		st:       storage,
		store:    stor,
		cert:     cert,
		source:   pack.ManualSource{},
	}

	s.CreateDefaultLinks(cfg.License)

	log.Printf("License server running on port %d [readonly %t]", cfg.LcpServer.Port, readonly)
	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	// methods related to EPUB encrypted content

	contentRoutesPathPrefix := "/contents"
	contentRoutes := sr.R.PathPrefix(contentRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.handleFunc(sr.R, contentRoutesPathPrefix, LCPCtrl.ListContents).Methods("GET")

	// get encrypted content by content id (a uuid)
	s.handleFunc(contentRoutes, "/{content_id}", LCPCtrl.GetContent).Methods("GET")
	// get all licenses associated with a given content
	s.handlePrivateFunc(contentRoutes, "/{content_id}/licenses", LCPCtrl.ListLicensesForContent, basicAuth).Methods("GET")

	if !readonly {
		s.handleFunc(contentRoutes, "/{name}", LCPCtrl.StoreContent).Methods("POST")
		// put content to the storage
		s.handlePrivateFunc(contentRoutes, "/{content_id}", LCPCtrl.AddContent, basicAuth).Methods("PUT")
		// generate a license for given content
		s.handlePrivateFunc(contentRoutes, "/{content_id}/license", LCPCtrl.GenerateLicense, basicAuth).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.handlePrivateFunc(contentRoutes, "/{content_id}/licenses", LCPCtrl.GenerateLicense, basicAuth).Methods("POST")
		// generate a licensed publication
		s.handlePrivateFunc(contentRoutes, "/{content_id}/publication", LCPCtrl.GenerateLicensedPublication, basicAuth).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.handlePrivateFunc(contentRoutes, "/{content_id}/publications", LCPCtrl.GenerateLicensedPublication, basicAuth).Methods("POST")
	}

	// methods related to licenses

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.handlePrivateFunc(sr.R, licenseRoutesPathPrefix, LCPCtrl.ListLicenses, basicAuth).Methods("GET")
	// get a license
	s.handlePrivateFunc(licenseRoutes, "/{license_id}", LCPCtrl.GetLicense, basicAuth).Methods("GET")
	s.handlePrivateFunc(licenseRoutes, "/{license_id}", LCPCtrl.GetLicense, basicAuth).Methods("POST")
	// get a licensed publication via a license id
	s.handlePrivateFunc(licenseRoutes, "/{license_id}/publication", LCPCtrl.GetLicensedPublication, basicAuth).Methods("POST")
	if !readonly {
		// update a license
		s.handlePrivateFunc(licenseRoutes, "/{license_id}", LCPCtrl.UpdateLicense, basicAuth).Methods("PATCH")
	}

	s.source.Feed(packager.Incoming)
	return s
}
