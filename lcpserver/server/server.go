package lcpserver

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/technoweenie/grohl"

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
	router   *mux.Router
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

func New(bindAddr string, tplPath string, readonly bool, idx *index.Index, st *storage.Store, lst *license.Store, cert *tls.Certificate, packager *pack.Packager, basicAuth *auth.BasicAuth) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler:      r,
			Addr:         bindAddr,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		},
		readonly: readonly,
		idx:      idx,
		st:       st,
		lst:      lst,
		cert:     cert,
		router:   r,
		source:   pack.ManualSource{},
	}

	s.source.Feed(packager.Incoming)

	hfs := http.FileServer(http.Dir(tplPath))
	r.PathPrefix("/manage").Handler(hfs)

	//API following spec
	//CONTENTS
	s.handleFunc("/contents", apilcp.ListContents).Methods("GET") //method supported, not in spec
	s.handleFunc("/contents/", apilcp.ListContents).Methods("GET")
	s.handleFunc("/contents/{key}", apilcp.GetContent).Methods("GET")                                         //method supported, not in spec
	s.handlePrivateFunc("/contents/{key}/licenses", apilcp.ListLicensesForContent, basicAuth).Methods("GET")  // list licenses for content, additional get params {page?,per_page?}
	s.handlePrivateFunc("/contents/{key}/licenses/", apilcp.ListLicensesForContent, basicAuth).Methods("GET") // idem
	//LICENSES
	s.handlePrivateFunc("/licenses", apilcp.ListLicenses, basicAuth).Methods("GET")     // list licenses, additional get params {page?,per_page?}
	s.handlePrivateFunc("/licenses/", apilcp.ListLicenses, basicAuth).Methods("GET")    // idem
	s.handlePrivateFunc("/licenses/{key}", apilcp.GetLicense, basicAuth).Methods("GET") //return existing license

	if !readonly {
		s.handlePrivateFunc("/contents/{key}", apilcp.AddContent, basicAuth).Methods("PUT") //lcp spec store data resulting from external encryption
		s.handleFunc("/contents/{name}", apilcp.StoreContent).Methods("POST")               //lcp spec encrypt & store epub file (in BODY)
		s.handlePrivateFunc("/contents/{key}/licenses", apilcp.GenerateLicense, basicAuth).Methods("POST")
		s.handlePrivateFunc("/contents/{key}/publications/", apilcp.GenerateProtectedPublication, basicAuth).Methods("POST")
		s.handlePrivateFunc("/contents/{key}/publications", apilcp.GenerateProtectedPublication, basicAuth).Methods("POST")
		//LICENSES
		s.handlePrivateFunc("/licenses/{key}", apilcp.UpdateLicense, basicAuth).Methods("PATCH") //update license (rights, other)
	}

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler) //handle all other requests 404

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilcp.Server)
type HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s apilcp.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path})
		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}

func (s *Server) handlePrivateFunc(route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		var username string
		if username = authenticator.CheckAuth(r); username == "" {
			grohl.Log(grohl.Data{"error": "Unauthorized", "method": r.Method, "path": r.URL.Path})
			w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "User or password do not match!"}, http.StatusUnauthorized)
			return
		}
		grohl.Log(grohl.Data{"user": username, "method": r.Method, "path": r.URL.Path})

		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
