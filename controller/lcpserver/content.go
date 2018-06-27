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

package lcpserver

import (
	"bytes"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	goHttp "net/http"
	"os"
)

// StoreContent stores content in the storage
// the content name is given in the url (name)
// a temporary file is created, then deleted after the content has been stored
//
func StoreContent(server http.IServer, req *goHttp.Request, param ParamName) (*string, error) {
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// defering close
	defer req.Body.Close()

	size, payload, err := writeRequestFileToTemp(bytes.NewReader(reqBody))
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	defer cleanupTemp(payload)

	result := server.Source().Post(pack.NewTask(param.Name, payload, size))
	if result.Error != nil {
		return nil, http.Problem{Detail: result.Error.Error(), Status: http.StatusBadRequest}
	}

	resultId := &result.Id
	server.LogInfo("Created : %s", resultId)

	return resultId, http.Problem{Status: http.StatusCreated}
}

// AddContent adds content to the storage
// lcp spec : store data resulting from an external encryption
// PUT method with PAYLOAD : LcpPublication in json format
// content_id is also present in the url.
// if contentID is different , url key overrides the content id in the json payload
// this method adds the <protected_content_location>  in the store (of encrypted files)
// and the key in the database in order to create the licenses

//Payload: (json) {content-id, content-encryption-key, protected-content-location, protected-content-length, protected-content-sha256, protected-content-disposition}

func AddContent(server http.IServer, publication *http.AuthorizationAndLcpPublication) (*model.Content, error) {
	//server.LogInfo("Payload %#v\nParam %#v", publication, publication.ContentId)
	if publication.ContentId == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}

	// open the encrypted file from the path given in the json payload
	file, err := os.Open(publication.Output)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	defer file.Close()

	// TODO : seems this route is used for both create and update - so update is going to replace the file without any checks.
	// TODO : shouldn't be this the last step, after operating database?
	// add the file to the storage, named from contentID
	_, err = server.Storage().Add(publication.ContentId, file)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	// check row in database
	content, foundErr := server.Store().Content().Get(publication.ContentId)
	content.Id = publication.ContentId
	content.EncryptionKey = publication.ContentKey
	// default values
	content.Location = ""
	content.Length = -1
	content.Sha256 = ""

	if publication.ContentDisposition != nil {
		content.Location = *publication.ContentDisposition
	}

	if publication.Size != nil {
		content.Length = *publication.Size
	}

	if publication.Checksum != nil {
		content.Sha256 = *publication.Checksum
	}

	//todo? check hash & length
	code := http.StatusCreated
	if foundErr == gorm.ErrRecordNotFound {
		// insert into database
		err = server.Store().Content().Add(content) // err gets checked below
	} else {
		//TODO : in which conditions this happens?
		// update encryption key for content.Id = publication.ContentId
		err = server.Store().Content().Update(content) // err gets checked below
		code = http.StatusOK
	}

	if err != nil { //db error
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	return content, http.Problem{Status: code}
}

// ListContents lists the content in the storage index
//
func ListContents(server http.IServer) (model.ContentCollection, error) {
	contents, err := server.Store().Content().List()
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	return contents, nil
}

// GetContent fetches and returns an encrypted content file
// selected by it content id (uuid)
//
func GetContent(server http.IServer, param ParamContentId) (io.ReadCloser, error) {
	if param.ContentID == "" {
		return nil, http.Problem{Detail: "The content id must be set in the url", Status: http.StatusBadRequest}
	}
	// get the content id from the calling url
	contentID := param.ContentID
	content, err := server.Store().Content().Get(contentID)
	if err != nil { //item probably not found
		if err == gorm.ErrRecordNotFound {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		} else {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	// check the existence of the file
	item, err := server.Storage().Get(contentID)
	if err != nil {
		// item not found ?
		if err == filestor.ErrNotFound {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		} else {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	nonErr := http.Problem{HttpHeaders: make(map[string][]string)}

	// opens the file
	contentReadCloser, err := item.Contents()
	if err != nil {
		//file probably not found
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// set headers
	nonErr.HttpHeaders.Add(http.HdrContentDisposition, "attachment; filename="+content.Location)
	nonErr.HttpHeaders.Add(http.HdrContentType, epub.ContentTypeEpub)
	nonErr.HttpHeaders.Add(http.HdrContentLength, fmt.Sprintf("%d", content.Length))
	// closing the io.ReadCloser is done in the server
	return contentReadCloser, nonErr
}
