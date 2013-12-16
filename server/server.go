package server

import (
  "net/http"
  "github.com/gorilla/mux"
  "github.com/demarque/lcpserve/index"
  "github.com/demarque/lcpserve/storage"
  "github.com/demarque/lcpserve/server/api"
  "html/template"
  "log"
)

type Server struct {
  http.Server
  idx *index.Index
  st *storage.Store
  router *mux.Router
}

func (s *Server) Store() storage.Store {
  return *s.st
}

func (s *Server) Index() index.Index {
  return *s.idx
}

func New(bindAddr string, idx *index.Index, st *storage.Store) *Server {
  r := mux.NewRouter()
  s := &Server{
    Server: http.Server {
      Handler: r,
      Addr: bindAddr,
    },
    idx: idx,
    st: st,
    router: r,
  }
  manageIndex, err := template.ParseFiles("static/manage/index.html")
  if err != nil {
    panic(err)
  }
  r.HandleFunc("/manage/", func(w http.ResponseWriter, r *http.Request) {
    manageIndex.Execute(w, map[string]interface{}{})
  })
  r.Handle("/manage/{file}", http.FileServer(http.Dir("static")))


  r.Handle("/files/{file}", http.StripPrefix("/files/", http.FileServer(http.Dir("files"))))
  s.handleFunc("/api/store/{name}", api.StorePackage).Methods("POST")
  r.Handle("/", http.NotFoundHandler())

  return s
}

type HandlerFunc func(w http.ResponseWriter, r * http.Request, s api.Server)

func (s * Server) handleFunc(route string, fn HandlerFunc) *mux.Route {
  return s.router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route called: %s", route)
    // Add CORS
    w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Add("Access-Control-Allow-Origin", "*")
    fn(w, r, s)
  })
}
