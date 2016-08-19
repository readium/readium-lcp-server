package lsdserver

import (
	"html/template"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/lsdserver/api"
	"github.com/readium/readium-lcp-server/transactions"
	"github.com/technoweenie/grohl"

	"net/http"
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

func New(bindAddr string, tplPath string, readonly bool, hist *history.History, trns *transactions.Transactions) *Server {
	r := mux.NewRouter()
	s := &Server{
		Server: http.Server{
			Handler: r,
			Addr:    bindAddr,
		},
		readonly: readonly,
		router:   r,
		hist:     *hist,
		trns:     *trns,
	}

	manageIndex, err := template.ParseFiles(filepath.Join(tplPath, "/manage/index.html"))
	if err != nil {
		panic(err)
	}

	r.HandleFunc("/manage/", func(w http.ResponseWriter, r *http.Request) {
		manageIndex.Execute(w, map[string]interface{}{})
	})

	s.handleFunc("/licenses/{key}/status", api.GenerateLicenseStatusDocument).Methods("POST")
	s.handleFunc("/licenses/", api.CreateLicenseStatusDocument).Methods("PUT")

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
