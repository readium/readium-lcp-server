package lcpserver

import (
	"crypto/tls"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/technoweenie/grohl"

	"html/template"
	"net/http"
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

func New(bindAddr string, tplPath string, readonly bool, idx *index.Index, st *storage.Store, lst *license.Store, cert *tls.Certificate, packager *pack.Packager) *Server {
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

	manageIndex, err := template.ParseFiles(filepath.Join(tplPath, "/manage/index.html"))
	if err != nil {
		panic(err)
	}
	r.HandleFunc("/manage/", func(w http.ResponseWriter, r *http.Request) {
		manageIndex.Execute(w, map[string]interface{}{})
	})
	r.Handle("/manage/{file}", http.FileServer(http.Dir("static")))

	r.Handle("/files/{file}", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))

	if !readonly {
		s.handleFunc("/api/store/{name}", apilcp.StoreContent).Methods("POST")
	}

	//API following spec
	//CONTENTS
	s.handleFunc("/contents", apilcp.ListContents).Methods("GET")                           //method supported, not in spec
	s.handleFunc("/contents/", apilcp.ListContents).Methods("GET")                          //method supported, not in spec
	s.handleFunc("/contents/{key}", apilcp.AddContent).Methods("PUT")                       //lcp spec store data resulting from external encryption
	s.handleFunc("/contents/{name}", apilcp.StoreContent).Methods("POST")                   //lcp spec encrypt & store epub file (in BODY)
	s.handleFunc("/contents/{key}/licenses", apilcp.ListLicensesForContent).Methods("GET")  // list licenses for content, additional get params {page?,per_page?}
	s.handleFunc("/contents/{key}/licenses/", apilcp.ListLicensesForContent).Methods("GET") // idem
	s.handleFunc("/contents/{key}/licenses", apilcp.GenerateLicense).Methods("POST")
	s.handleFunc("/contents/{key}/publications", apilcp.GenerateProtectedPublication).Methods("POST")

	//LICENSES
	s.handleFunc("/licenses", apilcp.ListLicenses).Methods("GET")          // list licenses, additional get params {page?,per_page?}
	s.handleFunc("/licenses/", apilcp.ListLicenses).Methods("GET")         // idem
	s.handleFunc("/licenses/{key}", apilcp.GetLicense).Methods("GET")      //return existing license
	s.handleFunc("/licenses/{key}", apilcp.UpdateLicense).Methods("PATCH") //update license (rights, other)

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler) //handle all other requests 404

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s apilcp.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path})

		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
