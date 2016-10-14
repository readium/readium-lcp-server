package lcpserver

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/technoweenie/grohl"
	"github.com/urfave/negroni"
	
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
)

type Server struct {
	http.Server
	readonly bool
	idx      *index.Index
	st       *storage.Store
	lst      *license.Store
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

func ExtraLogger(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path})

// before	
	next(rw, r)
// after

	// noop
}

func CORSHeaders(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	grohl.Log(grohl.Data{"CORS": "yes"})
	rw.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Add("Access-Control-Allow-Origin", "*")

// before	
	next(rw, r)
// after

	// noop
}

func New(bindAddr string, tplPath string, readonly bool, idx *index.Index, st *storage.Store, lst *license.Store, cert *tls.Certificate, packager *pack.Packager, basicAuth *auth.BasicAuth) *Server {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler) //handle all other requests 404

	// this demonstrates a panic report
	r.HandleFunc("/panic", func(w http.ResponseWriter, req *http.Request) {
		panic("just testing. no worries.")
	})

	//n := negroni.Classic() == negroni.New(negroni.NewRecovery(), negroni.NewLogger(), negroni.NewStatic(...))
	n := negroni.New()

	// possibly useful middlewares:
	// https://github.com/jeffbmartinez/delay

	//https://github.com/urfave/negroni#recovery
	recovery := negroni.NewRecovery()
	recovery.PrintStack = true
	recovery.ErrorHandlerFunc = problem.PanicReport
	n.Use(recovery)

	//https://github.com/urfave/negroni#logger
	n.Use(negroni.NewLogger())

	n.Use(negroni.HandlerFunc(ExtraLogger))
	
	//https://github.com/urfave/negroni#static
	n.Use(negroni.NewStatic(http.Dir(tplPath)))

	n.Use(negroni.HandlerFunc(CORSHeaders))
	// Does not insert CORS headers as intended, depends on Origin check in the HTTP request...we want the same headers, always.
	// IMPORT "github.com/rs/cors"
	// //https://github.com/rs/cors#parameters
	// c := cors.New(cors.Options{
	// 	AllowedOrigins: []string{"*"},
	// 	AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
	// 	Debug: true,
	// })
	// n.Use(c)

	s := &Server{
		Server: http.Server{
			Handler:      n,
			Addr:         bindAddr,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		readonly: readonly,
		idx:      idx,
		st:       st,
		lst:      lst,
		cert:     cert,
		source:   pack.ManualSource{},
	}


	contentRoutes := r.PathPrefix("/contents").Subrouter().StrictSlash(false)
	
	// note that "/contents" would 301-redirect to "/contents/" if StrictSlash(true)
	// note that "/contents/KEY/" would 301-redirect to "/contents/KEY" if StrictSlash(true)
	
	s.handleFunc(r, "/contents", apilcp.ListContents).Methods("GET") // annoyingly redundant, but we must add this route "manually" as the PathPrefix() with StrictSlash(false) dictates 
	s.handleFunc(contentRoutes, "/", apilcp.ListContents).Methods("GET")

	s.handleFunc(contentRoutes, "/{key}", apilcp.GetContent).Methods("GET")
	s.handlePrivateFunc(contentRoutes, "/{key}/licenses", apilcp.ListLicensesForContent, basicAuth).Methods("GET")
	if !readonly {
		s.handleFunc(contentRoutes, "/{name}", apilcp.StoreContent).Methods("POST")
		s.handlePrivateFunc(contentRoutes, "/{key}", apilcp.AddContent, basicAuth).Methods("PUT")
		s.handlePrivateFunc(contentRoutes, "/{key}/licenses", apilcp.GenerateLicense, basicAuth).Methods("POST")
		s.handlePrivateFunc(contentRoutes, "/{key}/publications", apilcp.GenerateProtectedPublication, basicAuth).Methods("POST")
	}

	licenseRoutes := r.PathPrefix("/licenses").Subrouter().StrictSlash(false)

	s.handlePrivateFunc(r, "/licenses", apilcp.ListLicenses, basicAuth).Methods("GET") // annoyingly redundant, but we must add this route "manually" as the PathPrefix() with StrictSlash(false) dictates
	s.handlePrivateFunc(licenseRoutes, "/", apilcp.ListLicenses, basicAuth).Methods("GET")

	s.handlePrivateFunc(licenseRoutes, "/{key}", apilcp.GetLicense, basicAuth).Methods("GET")
	if !readonly {
		s.handlePrivateFunc(licenseRoutes, "/{key}", apilcp.UpdateLicense, basicAuth).Methods("PATCH")
	}

	n.UseHandler(r)

	s.source.Feed(packager.Incoming)
	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilcp.Server)
func (s *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, s)
	})
}

type HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s apilcp.Server)
func (s *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		var username string
		if username = authenticator.CheckAuth(r); username == "" {
			grohl.Log(grohl.Data{"error": "Unauthorized"})
			w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "User or password do not match!"}, http.StatusUnauthorized)
			return
		}
		grohl.Log(grohl.Data{"user": username})
		fn(w, r, s)
	})
}
