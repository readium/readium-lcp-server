package apilcp

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/problem"
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

// struct for communication with lcp-server
type LcpPublication struct {
	ContentId    string `json:"content-id"`
	ContentKey   []byte `json:"content-encryption-key"`
	Output       string `json:"protected-content-location"`
	ErrorMessage string `json:"error"`
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

func StoreContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	size, f, err := writeRequestFileToTemp(r.Body)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	defer cleanupTemp(f)

	t := pack.NewTask(vars["name"], f, size)
	result := s.Source().Post(t)

	if result.Error != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: result.Error.Error()}, http.StatusBadRequest)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(result.Id)
}

// AddContent()
// lcp spec : store data resulting from an external encryption
// PUT method with PAYLOAD : LcpPublication in json format
// content_id is also present in also url.
// if contentId is different , url key overrides the contentId in the json payload
// this method adds ths <protected_content_location>  in the store (of encrypted files)
// and the needed key in the database in order to create the licenses
func AddContent(w http.ResponseWriter, r *auth.AuthenticatedRequest, s Server) {
	request := &r.Request

	vars := mux.Vars(request)
	decoder := json.NewDecoder(request.Body)

	var publication LcpPublication
	err := decoder.Decode(&publication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	contentId := vars["key"]
	if contentId != publication.ContentId {
		publication.ContentId = contentId
	}
	//read encrypted file from reference
	file, err := os.Open(publication.Output)
	if err != nil {
		problem.Error(w, request, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	//and add file to storage
	var storageItem storage.Item
	storageItem, err = s.Store().Add(publication.ContentId, file)
	if err != nil {
		problem.Error(w, request, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	var c index.Content
	// insert row in database if key does not exist
	c, err = s.Index().Get(publication.ContentId)
	c.EncryptionKey = publication.ContentKey
	c.Location = storageItem.Key()
	code := http.StatusCreated
	if err == index.NotFound { //insert into database
		c.Id = publication.ContentId
		err = s.Index().Add(c)
	} else { //update encryption key for c.Id = publication.ContentId
		err = s.Index().Update(c)
		code = http.StatusFound
	}
	if err != nil { //db not updated
		problem.Error(w, request, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	return
	//json.NewEncoder(w).Encode(publication.ContentId)

}

func ListContents(w http.ResponseWriter, r *http.Request, s Server) {
	fn := s.Index().List()
	contents := make([]index.Content, 0)

	for it, err := fn(); err == nil; it, err = fn() {
		contents = append(contents, it)
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(contents)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	problem.Error(w, r, problem.Problem{Type: "about:blank"}, http.StatusNotFound)
}
