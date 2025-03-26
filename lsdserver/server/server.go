// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package lsdserver

import (
	"net/http"
	"time"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	licensestatuses "github.com/readium/readium-lcp-server/license_statuses"
	apilsd "github.com/readium/readium-lcp-server/lsdserver/api"
	"github.com/readium/readium-lcp-server/transactions"
)

type Server struct {
	http.Server
	readonly  bool
	goofyMode bool
	lst       licensestatuses.LicenseStatuses
	trns      transactions.Transactions
}

func (s *Server) LicenseStatuses() licensestatuses.LicenseStatuses {
	return s.lst
}

func (s *Server) Transactions() transactions.Transactions {
	return s.trns
}

func (s *Server) GoofyMode() bool {
	return s.goofyMode
}

func New(bindAddr string, readonly bool, goofyMode bool, lst *licensestatuses.LicenseStatuses, trns *transactions.Transactions, basicAuth *auth.BasicAuth) *Server {

	sr := api.CreateServerRouter("")

	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           bindAddr,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		readonly:  readonly,
		lst:       *lst,
		trns:      *trns,
		goofyMode: goofyMode,
	}

	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	// Ping endpoint
	s.handleFunc(sr.R, "/ping", apilsd.Ping).Methods("GET")

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.handlePrivateFunc(sr.R, licenseRoutesPathPrefix, apilsd.FilterLicenseStatuses, basicAuth).Methods("GET")

	s.handleFunc(licenseRoutes, "/{key}/status", apilsd.GetLicenseStatusDocument).Methods("GET")
	s.handleFunc(licenseRoutes, "/{key}", apilsd.GetFreshLicense).Methods("GET")

	s.handlePrivateFunc(licenseRoutes, "/{key}/registered", apilsd.ListRegisteredDevices, basicAuth).Methods("GET")
	if !readonly {
		s.handleFunc(licenseRoutes, "/{key}/register", apilsd.RegisterDevice).Methods("POST")
		s.handleFunc(licenseRoutes, "/{key}/return", apilsd.LendingReturn).Methods("PUT")
		s.handleFunc(licenseRoutes, "/{key}/renew", apilsd.LendingRenewal).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/{key}/status", apilsd.LendingCancellation, basicAuth).Methods("PATCH")
		s.handlePrivateFunc(licenseRoutes, "/{key}/extend", apilsd.ExtendSubscription, basicAuth).Methods("PUT")

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
