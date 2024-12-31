// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilcp

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/storage"
)

// Server groups functions used by the lcp server
type Server interface {
	Store() storage.Store
	Index() index.Index
	Licenses() license.Store
	Certificate() *tls.Certificate
	Source() *pack.ManualSource
}

// Encrypted is used for communication with the License Server
type Encrypted struct {
	ContentID   string `json:"content-id"`
	ContentKey  []byte `json:"content-encryption-key"`
	StorageMode int    `json:"storage-mode"`
	Output      string `json:"protected-content-location"`
	FileName    string `json:"protected-content-disposition"`
	Size        int64  `json:"protected-content-length"`
	Checksum    string `json:"protected-content-sha256"`
	ContentType string `json:"protected-content-type,omitempty"`
}

const (
	Storage_none = 0
	Storage_s3   = 1
	Storage_fs   = 2
)

func writeRequestFileToTemp(r io.Reader) (int64, *os.File, error) {
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "readium-lcp")
	if err != nil {
		return 0, file, err
	}

	n, err := io.Copy(file, r)

	// Rewind to the beginning of the file
	file.Seek(0, 0)

	return n, file, err
}

func cleanupTempFile(f *os.File) {
	if f == nil {
		return
	}
	f.Close()
	os.Remove(f.Name())
}

// StoreContent stores content passed through the request body into the storage.
// The content name is given in the url (name)
// A temporary file is created, then deleted after the content has been stored.
// This function is using an async task.
func StoreContent(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)

	size, f, err := writeRequestFileToTemp(r.Body)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	defer cleanupTempFile(f)

	t := pack.NewTask(vars["name"], f, size)
	result := s.Source().Post(t)

	if result.Error != nil {
		problem.Error(w, r, problem.Problem{Detail: result.Error.Error()}, http.StatusBadRequest)
		return
	}

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(result.ID)
}

// AddContent adds content to the storage
// lcp spec : store data resulting from an external encryption
// PUT method with PAYLOAD : encrypted publication in json format
// This method adds an encrypted file to a store
// and adds the corresponding decryption key to the database.
// The content_id is taken from  the url.
// The input file is then deleted.
func AddContent(w http.ResponseWriter, r *http.Request, s Server) {

	// parse the json payload
	vars := mux.Vars(r)
	decoder := json.NewDecoder(r.Body)
	var encrypted Encrypted
	err := decoder.Decode(&encrypted)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	// get the content ID in the url
	contentID := vars["content_id"]
	if contentID == "" {
		problem.Error(w, r, problem.Problem{Detail: "The content id must be set in the url"}, http.StatusBadRequest)
		return
	}

	// add a log
	logging.Print("Add publication " + contentID)

	// if the encrypted publication has not been already stored by lcpencrypt
	if encrypted.StorageMode == Storage_none {

		// open the encrypted file, use its full path
		file, err := getAndOpenFile(encrypted.Output)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		// the input file will be deleted when the function returns
		defer cleanupTempFile(file)

		// add the file to the storage, named by contentID, without file extension
		_, err = s.Store().Add(contentID, file)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	}

	// insert a row in the database if the content id does not already exist
	// or update the database with new information if the content id already exists
	var c index.Content
	c, err = s.Index().Get(contentID)
	// err checked later ...
	c.EncryptionKey = encrypted.ContentKey
	// the Location field contains either the file name (useful during download)
	// or the storage URL of the encrypted, depending the storage mode.
	if encrypted.StorageMode != Storage_none {
		c.Location = encrypted.Output
	} else {
		c.Location = encrypted.FileName
	}
	c.Length = encrypted.Size
	c.Sha256 = encrypted.Checksum
	c.Type = encrypted.ContentType

	code := http.StatusCreated
	if err == index.ErrNotFound { //insert into database
		c.ID = contentID
		err = s.Index().Add(c)
		// the content id was found in the database
	} else { //update the encryption key for c.ID = encrypted.ContentID
		err = s.Index().Update(c)
		code = http.StatusOK

		if err == nil {
			log.Println("Update all license timestamps associated with this publication")
			err = s.Licenses().TouchByContentID(contentID) // update all licenses update timestamps
		}

	}
	if err != nil { //if db not updated
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// set the response http code
	w.WriteHeader(code)
}

// ListContents lists the content in the storage index
func ListContents(w http.ResponseWriter, r *http.Request, s Server) {

	fn := s.Index().List()
	contents := make([]index.Content, 0)

	var razkey []byte // in a list, we don't return the encryption key.
	for it, err := fn(); err == nil; it, err = fn() {
		it.EncryptionKey = razkey
		contents = append(contents, it)
	}

	// add a log
	logging.Print("List publications, total " + strconv.Itoa(len(contents)))

	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	err := enc.Encode(contents)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

// GetContentInfo returns information about the encrypted content,
// especially the encryption key.
// Used by the encryption utility when the file to encrypt is an update of an existing encrypted publication.
func GetContentInfo(w http.ResponseWriter, r *http.Request, s Server) {
	// get the content id from the calling url
	vars := mux.Vars(r)
	contentID := vars["content_id"]

	// add a log
	logging.Print("Get content info " + contentID)

	// get the info
	content, err := s.Index().Get(contentID)
	if err != nil { //item probably not found
		if err == index.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}

	// return the info
	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	err = enc.Encode(content)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

// GetContentFile fetches and returns an encrypted content file
// selected by it content id (uuid)
// This should be called only if the License Server stores the file.
// If it is not the case, the file should be fetched from a standard web server
func GetContentFile(w http.ResponseWriter, r *http.Request, s Server) {

	// get the content id from the calling url
	vars := mux.Vars(r)
	contentID := vars["content_id"]

	// add a log
	logging.Print("Fetch content " + contentID)

	content, err := s.Index().Get(contentID)
	if err != nil { //item probably not found
		if err == index.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}

	// check the existence of the file
	item, err := s.Store().Get(contentID)
	if err != nil { //item probably not found
		if err == storage.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: "Storage:" + err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: "Storage:" + err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}
	// opens the file
	contentReadCloser, err := item.Contents()
	if err != nil { //file probably not found
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusBadRequest)
		return
	}

	defer contentReadCloser.Close()

	// set headers
	// If this function is called for a file stored by the encrypting tool, we have to provide a sensible
	// Content-Disposition header, to be used as file name after download.
	hasPubLink, err := isURL(content.Location)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: "Content Location:" + err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	var filename string
	if hasPubLink {
		// we have not stored the original file name, therefore we use the content id.
		filename = content.ID
	} else {
		// in the initial version of the server, the filename was in this field.
		filename = content.Location
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", content.Type)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", content.Length))

	// returns the content of the file to the caller
	io.Copy(w, contentReadCloser)
}

// DeleteContent deletes a record
func DeleteContent(w http.ResponseWriter, r *http.Request, s Server) {

	// get the content id from the calling url
	vars := mux.Vars(r)
	contentID := vars["content_id"]

	// add a log
	logging.Print("Delete publication " + contentID)

	err := s.Index().Delete(contentID)
	if err != nil { //item probably not found
		if err == index.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: "Index:" + err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}
	// set the response http code
	w.WriteHeader(http.StatusOK)

}

// getAndOpenFile opens a file from a path, or downloads then opens it if its location is a URL
func getAndOpenFile(filePathOrURL string) (*os.File, error) {

	isURL, err := isURL(filePathOrURL)
	if err != nil {
		return nil, err
	}

	if isURL {
		return downloadAndOpenFile(filePathOrURL)
	}

	return os.Open(filePathOrURL)
}

func downloadAndOpenFile(url string) (*os.File, error) {
	file, _ := os.CreateTemp("", "")
	fileName := file.Name()

	err := downloadFile(url, fileName)

	if err != nil {
		return nil, err
	}

	return os.Open(fileName)
}

func isURL(filePathOrURL string) (bool, error) {
	url, err := url.Parse(filePathOrURL)
	if err != nil {
		return false, errors.New("error parsing input string")
	}
	return url.Scheme == "http" || url.Scheme == "https", nil
}

func downloadFile(url string, targetFilePath string) error {
	out, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP response: %d %s when downloading %s", resp.StatusCode, resp.Status, url)
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Ping is a simple health check
func Ping(w http.ResponseWriter, r *http.Request, s Server) {
	w.WriteHeader(http.StatusOK)
}
