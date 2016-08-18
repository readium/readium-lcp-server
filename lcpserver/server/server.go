package lcpserver

import (
	"crypto/tls"
	"path/filepath"

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
		s.handleFunc("/api/store/{name}", api.StoreContent).Methods("POST")
	}

	//API following spec
	//CONTENTS
	s.handleFunc("/contents", api.ListContents).Methods("GET")                           //method supported, not in spec
	s.handleFunc("/contents/", api.ListContents).Methods("GET")                          //idem
	s.handleFunc("/contents/{key}", api.AddContent).Methods("PUT")                       //lcp spec store data resulting from external encryption
	s.handleFunc("/contents/{key}/licenses", api.ListLicensesForContent).Methods("GET")  // list licenses for content, additional get params {page?,per_page?}
	s.handleFunc("/contents/{key}/licenses/", api.ListLicensesForContent).Methods("GET") // idem
	s.handleFunc("/contents/{key}/licenses", api.GenerateLicense).Methods("POST")
	s.handleFunc("/contents/{key}/publications", api.GenerateProtectedPublication).Methods("POST")

	//LICENSES
	s.handleFunc("/licenses", api.ListLicenses).Methods("GET")                // list licenses, additional get params {page?,per_page?}
	s.handleFunc("/licenses/", api.ListLicenses).Methods("GET")               // idem
	s.handleFunc("/licenses/{key}", api.GetLicense).Methods("GET")            //return existing license
	s.handleFunc("/licenses/{key}", api.UpdateLicense).Methods("PUT")         //update license
	s.handleFunc("/licenses/{key}", api.UpdateRightsLicense).Methods("PATCH") //update license rights

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler)

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
