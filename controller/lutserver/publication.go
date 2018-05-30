/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package lutserver

import (
	"encoding/json"
	"log"
	"strconv"

	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
)

// GetPublications returns a list of publications
func GetPublications(server http.IServer, resp http.ResponseWriter, req *http.Request) (model.PublicationsCollection, error) {
	var page int64
	var perPage int64
	var err error

	if req.FormValue("page") != "" {
		page, err = strconv.ParseInt((req).FormValue("page"), 10, 32)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
		}
	} else {
		page = 1
	}

	if req.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((req).FormValue("per_page"), 10, 32)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
		}
	} else {
		perPage = 30
	}

	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}

	if page < 0 {
		return nil, http.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest}
	}
	// TODO : read noOfPublications
	noOfPublications := int64(0)
	pubs, err := server.Store().Publication().List(int(perPage), int(page))
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	if len(pubs) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		resp.Header().Set("Link", "</publications/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		resp.Header().Set("Link", "</publications/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	resp.Header().Set(http.HdrContentType, http.ContentTypeJson)

	enc := json.NewEncoder(resp)
	err = enc.Encode(pubs)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Set("Link", http.MakePaginationHeader("http://localhost:"+strconv.Itoa(server.Config().LcpServer.Port)+"/publications=", page+1, perPage, noOfPublications))
	return pubs, nonErr
}

// GetPublication returns a publication from its numeric id, given as part of the calling url
//
func GetPublication(server http.IServer, resp http.ResponseWriter, req *http.Request) (*model.Publication, error) {
	vars := mux.Vars(req)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		return nil, http.Problem{Detail: "The publication id must be an integer", Status: http.StatusBadRequest}
	}

	if pub, err := server.Store().Publication().Get(int64(id)); err == nil {
		enc := json.NewEncoder(resp)
		if err = enc.Encode(pub); err == nil {
			// send json of correctly encoded user info
			resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
			resp.WriteHeader(http.StatusOK)
			return nil, nil
		}
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
}

// CheckPublicationByTitle check if a publication with this title exist
func CheckPublicationByTitle(server http.IServer, resp http.ResponseWriter, req *http.Request) (*string, error) {
	var title string
	title = req.URL.Query()["title"][0]

	log.Println("Check publication stored with name " + string(title))

	if pub, err := server.Store().Publication().CheckByTitle(string(title)); err == nil {
		enc := json.NewEncoder(resp)
		if err = enc.Encode(pub); err == nil {
			// send json of correctly encoded user info
			resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
			resp.WriteHeader(http.StatusOK)
			return nil, nil
		}
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	} else {
		switch err {
		case gorm.ErrRecordNotFound:

			log.Println("No publication stored with name " + string(title))
			//	server.Error(w, r, s.DefaultSrvLang(), common.Problem{Detail: err.Error(),Status: http.StatusNotFound)

		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

		}
	}
	return nil, nil
}

// CreatePublication creates a publication in the database
func CreatePublication(server http.IServer, pub *model.Publication) (*string, error) {

	// get the path to the master file
	inputPath := path.Join(server.Config().FrontendServer.MasterRepository, pub.MasterFilename)

	if _, err := os.Stat(inputPath); err != nil {
		// the master file does not exist
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
	}

	contentDisposition := http.Slugify(pub.Title)
	// encrypt the EPUB File and send the content to the LCP server
	err := EncryptEPUB(inputPath, contentDisposition, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// add publication
	if err = server.Store().Publication().Add(pub); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	return nil, http.Problem{Status: http.StatusCreated, HttpHeaders: make(map[string][]string)}
}

// UploadEPUB creates a new EPUB file, namd after a file form parameter.
// a temp file is created then deleted.
//UploadEPUB creates a new EPUB file
func UploadEPUB(server http.IServer, resp http.ResponseWriter, req *http.Request) (*string, error) {
	//var pub store.Publication
	contentDisposition := http.Slugify(req.URL.Query()["title"][0])

	file, header, err := req.FormFile("file")

	tmpfile, err := ioutil.TempFile("", "example")

	if err != nil {
		fmt.Fprintln(resp, err)
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}

	defer os.Remove(tmpfile.Name())

	_, err = io.Copy(tmpfile, file)

	if err = tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	// encrypt the EPUB File and send the content to the LCP server
	if err = EncryptEPUB(tmpfile.Name(), contentDisposition, server); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(resp, "File uploaded successfully : ")
	fmt.Fprintf(resp, header.Filename)
	return nil, nil
}

// UpdatePublication updates an identified publication (id) in the database
func UpdatePublication(server http.IServer, pub *model.Publication, param ParamId) (*model.Publication, error) {
	id, err := strconv.Atoi(param.Id)
	if err != nil {
		// id is not a number
		return nil, http.Problem{Detail: "Plublication ID must be an integer", Status: http.StatusBadRequest}
	}

	// publication ok, id is a number, search publication to update
	if foundPub, err := server.Store().Publication().Get(int64(id)); err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	} else {
		// publication is found!
		if err := server.Store().Publication().Update(&model.Publication{ID: foundPub.ID, Title: pub.Title, Status: foundPub.Status}); err != nil {
			//update failed!
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		//database update ok
		return nil, http.Problem{Status: http.StatusOK}
	}
	return nil, nil
}

// DeletePublication removes a publication in the database
func DeletePublication(server http.IServer, resp http.ResponseWriter, req *http.Request) (*string, error) {
	vars := mux.Vars(req)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	publication, err := server.Store().Publication().Get(id)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
	}

	// delete the epub file from the master repository
	inputPath := path.Join(server.Config().FrontendServer.MasterRepository, publication.Title+".epub")

	if _, err := os.Stat(inputPath); err == nil {
		err = os.Remove(inputPath)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		}
	}

	if err = server.Store().Publication().Delete(id); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	// publication deleted from db
	resp.WriteHeader(http.StatusOK)
	return nil, nil
}
