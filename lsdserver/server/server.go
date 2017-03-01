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

package lsdserver

import (
	"net/http"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/license_statuses"
	"github.com/readium/readium-lcp-server/lsdserver/api"
	"github.com/readium/readium-lcp-server/transactions"
)

type Server struct {
	http.Server
	readonly bool
	lst      licensestatuses.LicenseStatuses
	trns     transactions.Transactions
}

func (s *Server) LicenseStatuses() licensestatuses.LicenseStatuses {
	return s.lst
}

func (s *Server) Transactions() transactions.Transactions {
	return s.trns
}

func New(bindAddr string, readonly bool, complianceMode bool, lst *licensestatuses.LicenseStatuses, trns *transactions.Transactions, basicAuth *auth.BasicAuth) *Server {

	sr := api.CreateServerRouter("")

	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           bindAddr,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		readonly: readonly,
		lst:      *lst,
		trns:     *trns,
	}

	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.handlePrivateFunc(sr.R, licenseRoutesPathPrefix, apilsd.FilterLicenseStatuses, basicAuth).Methods("GET")

	s.handleFunc(licenseRoutes, "/{key}/status", apilsd.GetLicenseStatusDocument).Methods("GET")

	if complianceMode {
		s.handleFunc(sr.R, "/compliancetest", apilsd.AddLogToFile).Methods("GET")
	}

	s.handlePrivateFunc(licenseRoutes, "/{key}/registered", apilsd.ListRegisteredDevices, basicAuth).Methods("GET")
	if !readonly {
		s.handleFunc(licenseRoutes, "/{key}/register", apilsd.RegisterDevice).Methods("POST")
		s.handleFunc(licenseRoutes, "/{key}/return", apilsd.LendingReturn).Methods("PUT")
		s.handleFunc(licenseRoutes, "/{key}/renew", apilsd.LendingRenewal).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/{key}/status", apilsd.CancelLicenseStatus, basicAuth).Methods("PATCH")

		s.handlePrivateFunc(sr.R, "/licenses", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
	}

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilsd.Server)

func (s *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, s)
	})
}

type HandlerPrivateFunc func(w http.ResponseWriter, r *http.Request, s apilsd.Server)

func (s *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerPrivateFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if api.CheckAuth(authenticator, w, r) {
			fn(w, r, s)
		}
	})
}
