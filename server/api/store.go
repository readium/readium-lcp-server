package api

import (
  "github.com/gorilla/mux"
  "github.com/demarque/lcpserve/epub"
  "github.com/demarque/lcpserve/pack"
  "github.com/demarque/lcpserve/storage"
  "github.com/demarque/lcpserve/index"
  "net/http"
  "bytes"
  "archive/zip"
  "io/ioutil"
  "log"
)

type Server interface {
  Store() storage.Store
  Index() index.Index
}

func StorePackage(w http.ResponseWriter, r *http.Request, s Server) {
  vars := mux.Vars(r)
  
  name := vars["name"]
  buf, err := ioutil.ReadAll(r.Body)
  if err != nil {
    log.Println("Error reading body")
    log.Println(err)
    w.WriteHeader(500)
    return
  }
  zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
  if err != nil {
    log.Println("Error opening zip")
    log.Println(err)
    w.WriteHeader(500)
    return
  }
  ep, err := epub.Read(*zr)
  if err != nil {
    log.Println("Error reading epub")
    log.Println(err)
    w.WriteHeader(500)
    return
  }
  out, encryptionKey, err := pack.Do(ep)
  if err != nil {
    log.Println("Error packing")
    log.Println(err)
    w.WriteHeader(500)
    return
  }

  output := new(bytes.Buffer)
  out.Write(output)
  _, err = s.Store().Add(name, output)
  if err != nil {
    log.Println("Error storing")
    log.Println(err)
    w.WriteHeader(500)
    return
  }
  err = s.Index().Add(index.Package{name, encryptionKey, name})
  if err != nil {
    log.Println("Error while adding to index")
    log.Println(err)
    w.WriteHeader(500)
    return
  }
  w.WriteHeader(200)
  w.Write([]byte(name))
}
