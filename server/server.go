package server

import (
	"crypto/tls"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/server/api"
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

func (s *Server) Encrypter() crypto.Encrypter {
	return crypto.NewAESCBCEncrypter()
}

func New(bindAddr string, tplPath string, readonly bool, idx *index.Index, st *storage.Store, lst *license.Store, cert *tls.Certificate, packager *pack.Packager) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler: r,
			Addr:    bindAddr,
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
		s.handleFunc("/api/store/{name}", api.StorePackage).Methods("POST")
	}
	s.handleFunc("/api/packages", api.ListPackages).Methods("GET")
	s.handleFunc("/api/packages/{key}/licenses", api.GrantLicense).Methods("POST")
	r.Handle("/", http.NotFoundHandler())

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s api.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		grohl.Log(grohl.Data{"path": r.URL.Path})

		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
