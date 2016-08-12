package lsdserver

import (
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/transactions"

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
	r.Handle("/", http.NotFoundHandler())

	return s
}
