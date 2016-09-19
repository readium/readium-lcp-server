package lsdserver

import (
	"time"

	"net/http"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/lsdserver/api"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/transactions"
	"github.com/technoweenie/grohl"
)

type Server struct {
	http.Server
	readonly bool
	router   *mux.Router
	hist     history.History
	trns     transactions.Transactions
}

func (s *Server) History() history.History {
	return s.hist
}

func (s *Server) Transactions() transactions.Transactions {
	return s.trns
}

func New(bindAddr string, readonly bool, hist *history.History, trns *transactions.Transactions, basicAuth *auth.BasicAuth) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler:      r,
			Addr:         bindAddr,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		},
		readonly: readonly,
		router:   r,
		hist:     *hist,
		trns:     *trns,
	}

	s.handleFunc("/licenses/{key}/status", apilsd.GetLicenseStatusDocument).Methods("GET")
	s.handleFunc("/licenses/{key}/register", apilsd.RegisterDevice).Methods("POST")
	s.handleFunc("/licenses/{key}/return", apilsd.LendingReturn).Methods("PUT")
	s.handleFunc("/licenses/{key}/renew", apilsd.LendingRenewal).Methods("PUT")
	s.handlePrivateFunc("/licenses", apilsd.FilterLicenseStatuses, basicAuth).Methods("GET")
	s.handlePrivateFunc("/licenses/{key}/registered", apilsd.ListRegisteredDevices, basicAuth).Methods("GET")
	s.handlePrivateFunc("/licenses/{key}/status", apilsd.CancelLicenseStatus, basicAuth).Methods("PATCH")
	s.handlePrivateFunc("/licenses", apilsd.CreateLicenseStatusDocument, basicAuth).Methods("PUT")

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler) //handle all other requests 404
	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilsd.Server)
type HandlerPrivateFunc func(w http.ResponseWriter, r *http.Request, s apilsd.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		grohl.Log(grohl.Data{"path": r.URL.Path})

		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
func (s *Server) handlePrivateFunc(route string, fn HandlerPrivateFunc, authenticator *auth.BasicAuth) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		var username string
		if username = authenticator.CheckAuth(r); username == "" {
			grohl.Log(grohl.Data{"error": "Unauthorized", "method": r.Method, "path": r.URL.Path})
			w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "User or password do not match!"}, http.StatusUnauthorized)
			return
		}
		grohl.Log(grohl.Data{"path": r.URL.Path})
		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
