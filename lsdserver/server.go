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

package lsdserver

import (
	"net/http"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/logger"
	"github.com/readium/readium-lcp-server/lsdserver/ctrl"
	"github.com/readium/readium-lcp-server/store"
	"strconv"
)

type (
	Server struct {
		http.Server
		readonly  bool
		goofyMode bool
		log       logger.StdLogger
		store     store.Store
		config    api.Configuration
	}
	HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilsd.IServer)

	HandlerPrivateFunc func(w http.ResponseWriter, r *http.Request, s apilsd.IServer)
)

func (s *Server) GoofyMode() bool {
	return s.goofyMode
}

func (s *Server) Store() store.Store {
	return s.store
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

func (s *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerPrivateFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if api.CheckAuth(authenticator, w, r) {
			fn(w, r, s)
		}
	})
}

func New(config api.Configuration,
	log logger.StdLogger,
	store store.Store,
	basicAuth *auth.BasicAuth) *Server {

	complianceMode := config.ComplianceMode
	parsedPort := strconv.Itoa(config.LsdServer.Port)
	readonly := config.LsdServer.ReadOnly
	// the server will behave strangely, to test the resilience of LCP compliant apps
	goofyMode := config.GoofyMode

	sr := api.CreateServerRouter("")

	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           ":" + parsedPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		log:       log,
		config:    config,
		readonly:  readonly,
		store:     store,
		goofyMode: goofyMode,
	}

	s.log.Printf("License status server running on port %d [readonly = %t]", config.LsdServer.Port, readonly)

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.handlePrivateFunc(sr.R, licenseRoutesPathPrefix, apilsd.FilterLicenseStatuses, basicAuth).Methods("GET")

	s.handleFunc(licenseRoutes, "/{key}/status", apilsd.GetLicenseStatusDocument).Methods("GET")

	if complianceMode {
		s.handleFunc(sr.R, "/compliancetest", apilsd.AddLogToFile).Methods("POST")
	}

	s.handlePrivateFunc(licenseRoutes, "/{key}/registered", apilsd.ListRegisteredDevices, basicAuth).Methods("GET")
	if !readonly {
		s.handleFunc(licenseRoutes, "/{key}/register", apilsd.RegisterDevice).Methods("POST")
		s.handleFunc(licenseRoutes, "/{key}/return", apilsd.LendingReturn).Methods("PUT")
		s.handleFunc(licenseRoutes, "/{key}/renew", apilsd.LendingRenewal).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/{key}/status", apilsd.LendingCancellation, basicAuth).Methods("PATCH")

		s.handlePrivateFunc(sr.R, "/licenses", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
	}

	return s
}
