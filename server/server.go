package server

import (
	"crypto/tls"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/jpbougie/lcpserve/index"
	"github.com/jpbougie/lcpserve/server/api"
	"github.com/jpbougie/lcpserve/storage"

	"html/template"
	"net/http"
)

type Server struct {
	http.Server
	idx    *index.Index
	st     *storage.Store
	router *mux.Router
	cert   *tls.Certificate
}

func (s *Server) Store() storage.Store {
	return *s.st
}

func (s *Server) Index() index.Index {
	return *s.idx
}

func (s *Server) Certificate() *tls.Certificate {
	return s.cert
}

func New(bindAddr string, tplPath string, idx *index.Index, st *storage.Store, cert *tls.Certificate) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler: r,
			Addr:    bindAddr,
		},
		idx:    idx,
		st:     st,
		cert:   cert,
		router: r,
	}

	manageIndex, err := template.ParseFiles(filepath.Join(tplPath, "/manage/index.html"))
	if err != nil {
		panic(err)
	}
	r.HandleFunc("/manage/", func(w http.ResponseWriter, r *http.Request) {
		manageIndex.Execute(w, map[string]interface{}{})
	})
	r.Handle("/manage/{file}", http.FileServer(http.Dir("static")))

	r.Handle("/files/{file}", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))
	s.handleFunc("/api/store/{name}", api.StorePackage).Methods("POST")
	s.handleFunc("/api/packages", api.ListPackages).Methods("GET")
	s.handleFunc("/api/packages/{key}/licenses", api.GrantLicense).Methods("POST")
	r.Handle("/", http.NotFoundHandler())

	return s
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request, s api.Server)

func (s *Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
	return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		// Add CORS
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		fn(w, r, s)
	})
}
