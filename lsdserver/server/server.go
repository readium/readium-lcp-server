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

func New(bindAddr string, readonly bool, lst *licensestatuses.LicenseStatuses, trns *transactions.Transactions, basicAuth *auth.BasicAuth) *Server {

	sr := api.CreateServerRouter("")
	
	s := &Server{
		Server: http.Server{
			Handler:      sr.N,
			Addr:         bindAddr,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		readonly: readonly,
		lst:      *lst,
		trns:     *trns,
	}

	licenseRoutes := sr.R.PathPrefix("/licenses").Subrouter().StrictSlash(false)

	// note that "/licenses" would 301-redirect to "/licenses/" if StrictSlash(true)
	// note that "/licenses/KEY/" would 301-redirect to "/licenses/KEY" if StrictSlash(true)
	
	s.handleFunc(licenseRoutes, "/{key}/status", apilsd.GetLicenseStatusDocument).Methods("GET")
	
	s.handlePrivateFunc(sr.R, "/licenses", apilsd.FilterLicenseStatuses, basicAuth).Methods("GET") // annoyingly redundant, but we must add this route "manually" as the PathPrefix() with StrictSlash(false) dictates
	s.handlePrivateFunc(licenseRoutes, "/", apilsd.FilterLicenseStatuses, basicAuth).Methods("GET")
	
	s.handlePrivateFunc(licenseRoutes, "/{key}/registered", apilsd.ListRegisteredDevices, basicAuth).Methods("GET")
	if !readonly {
		s.handleFunc(licenseRoutes, "/{key}/register", apilsd.RegisterDevice).Methods("POST")
		s.handleFunc(licenseRoutes, "/{key}/return", apilsd.LendingReturn).Methods("PUT")
		s.handleFunc(licenseRoutes, "/{key}/renew", apilsd.LendingRenewal).Methods("PUT")
		s.handlePrivateFunc(licenseRoutes, "/{key}/status", apilsd.CancelLicenseStatus, basicAuth).Methods("PATCH")

		s.handlePrivateFunc(sr.R, "/licenses", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT") // annoyingly redundant, but we must add this route "manually" as the PathPrefix() with StrictSlash(false) dictates
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
