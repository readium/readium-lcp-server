package api

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"

	"net/http"
)

type Server interface {
	Store() storage.Store
	Index() index.Index
	Licenses() license.Store
	Certificate() *tls.Certificate
	Source() *pack.ManualSource
}

func writeRequestFileToTemp(r io.Reader) (int64, *os.File, error) {
	dir := os.TempDir()
	file, err := ioutil.TempFile(dir, "readium-lcp")
	if err != nil {
		return 0, file, err
	}

	n, err := io.Copy(file, r)

	// Rewind to the beginning of the file
	file.Seek(0, 0)

	return n, file, err
}

func cleanupTemp(f *os.File) {
	if f == nil {
		return
	}
	f.Close()
	os.Remove(f.Name())
}

func StorePackage(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	size, f, err := writeRequestFileToTemp(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer cleanupTemp(f)

	t := pack.NewTask(vars["name"], f, size)
	result := s.Source().Post(t)

	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(result.Id)
}

func ListPackages(w http.ResponseWriter, r *http.Request, s Server) {
	fn := s.Index().List()
	packages := make([]index.Package, 0)

	for it, err := fn(); err == nil; it, err = fn() {
		packages = append(packages, it)
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(packages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

}
