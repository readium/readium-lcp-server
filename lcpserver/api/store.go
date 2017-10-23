// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package apilcp

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
)

type Server interface {
	Store() storage.Store
	Index() index.Index
	Licenses() license.Store
	Certificate() *tls.Certificate
	Source() *pack.ManualSource
}

// LcpPublication is a struct for communication with lcp-server
type LcpPublication struct {
	ContentId          string  `json:"content-id"`
	ContentKey         []byte  `json:"content-encryption-key"`
	Output             string  `json:"protected-content-location"`
	Size               *int64  `json:"protected-content-length,omitempty"`
	Checksum           *string `json:"protected-content-sha256,omitempty"`
	ContentDisposition *string `json:"protected-content-disposition,omitempty"`
	ErrorMessage       string  `json:"error"`
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

// StoreContent stores content in the storage
// the content name is given in the url (name)
// a temporary file is created, then deleted after the content has been stored
//
func StoreContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	size, f, err := writeRequestFileToTemp(r.Body)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	defer cleanupTemp(f)

	t := pack.NewTask(vars["name"], f, size)
	result := s.Source().Post(t)

	if result.Error != nil {
		problem.Error(w, r, problem.Problem{Detail: result.Error.Error()}, http.StatusBadRequest)
		return
	}

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(result.Id)
}

// AddContent adds content to the storage
// lcp spec : store data resulting from an external encryption
// PUT method with PAYLOAD : LcpPublication in json format
// content_id is also present in the url.
// if contentID is different , url key overrides the content id in the json payload
// this method adds the <protected_content_location>  in the store (of encrypted files)
// and the key in the database in order to create the licenses
func AddContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)

	// parse the json payload
	var publication LcpPublication
	err := decoder.Decode(&publication)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	// get the content ID in the url
	contentID := vars["content_id"]
	if contentID == "" {
		problem.Error(w, r, problem.Problem{Detail: "The content id must be set in the url"}, http.StatusBadRequest)
		return
	}
	// open the encrypted file from the path given in the json paylod
	file, err := os.Open(publication.Output)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// add the file to the storage, named from contentID
	_, err = s.Store().Add(contentID, file)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	var c index.Content
	// insert row in database if key does not exist
	c, err = s.Index().Get(contentID)
	c.EncryptionKey = publication.ContentKey
	if publication.ContentDisposition != nil {
		c.Location = *publication.ContentDisposition
	} else {
		c.Location = ""
	}

	if publication.Size != nil {
		c.Length = *publication.Size
	} else {
		c.Length = -1
	}

	if publication.Checksum != nil {
		c.Sha256 = *publication.Checksum
	} else {
		c.Sha256 = ""
	}
	//todo? check hash & length
	code := http.StatusCreated
	if err == index.NotFound { //insert into database
		c.Id = contentID
		err = s.Index().Add(c)
	} else { //update encryption key for c.Id = publication.ContentId
		err = s.Index().Update(c)
		code = http.StatusOK
	}
	if err != nil { //db not updated
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(code)

	return
	//json.NewEncoder(w).Encode(publication.ContentId)

}

// ListContents lists the content in the storage index
//
func ListContents(w http.ResponseWriter, r *http.Request, s Server) {
	fn := s.Index().List()
	contents := make([]index.Content, 0)

	for it, err := fn(); err == nil; it, err = fn() {
		contents = append(contents, it)
	}

	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	err := enc.Encode(contents)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

// GetContent fetches and returns an encrypted content file
// selected by it content id (uuid)
//
func GetContent(w http.ResponseWriter, r *http.Request, s Server) {
	// get the content id from the calling url
	vars := mux.Vars(r)
	contentID := vars["content_id"]
	content, err := s.Index().Get(contentID)
	if err != nil { //item probably not found
		if err == index.NotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	// check the existence of the file
	item, err := s.Store().Get(contentID)
	if err != nil { //item probably not found
		if err == storage.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	// opens the file
	contentReadCloser, err := item.Contents()
	defer contentReadCloser.Close()
	if err != nil { //file probably not found
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// set headers
	w.Header().Set("Content-Disposition", "attachment; filename="+content.Location)
	w.Header().Set("Content-Type", epub.ContentType_EPUB)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", content.Length))

	// returns the content of the file to the caller
	io.Copy(w, contentReadCloser)

	return

}
