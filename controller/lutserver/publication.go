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
	"bytes"
	"context"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/model"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// GetPublications returns a list of publications
func GetPublications(server http.IServer, param ParamPagination) (*views.Renderer, error) {
	noOfPublications, err := server.Store().Publication().Count()
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	// Pagination
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfPublications)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	var publications model.PublicationsCollection
	view := &views.Renderer{}
	if param.Filter != "" {
		view.AddKey("filter", param.Filter)
		noOfFilteredPublications, err := server.Store().Publication().FilterCount(param.Filter)
		if err != nil {
			return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
		}
		view.AddKey("filterTotal", noOfFilteredPublications)
		publications, err = server.Store().Publication().Filter(param.Filter, perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfFilteredPublications {
			view.AddKey("hasNextPage", true)
		}
		view.AddKey("noResults", noOfFilteredPublications == 0)
	} else {
		publications, err = server.Store().Publication().List(perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfPublications {
			view.AddKey("hasNextPage", true)
		}
		view.AddKey("noResults", noOfPublications == 0)
	}
	view.AddKey("publications", publications)
	view.AddKey("pageTitle", "Publications list")
	view.AddKey("total", noOfPublications)
	view.AddKey("currentPage", page+1)
	view.AddKey("perPage", perPage)
	view.Template("publications/index.html.got")
	return view, nil

}

// GetPublication returns a publication from its numeric id, given as part of the calling url
//
func GetPublication(server http.IServer, param ParamId) (*views.Renderer, error) {
	view := &views.Renderer{}
	if param.Id != "0" {
		id, err := strconv.Atoi(param.Id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
		}
		publication, err := server.Store().Publication().Get(int64(id))
		if err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}

			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
		view.AddKey("publication", publication)
		view.AddKey("pageTitle", "Edit publication")
	} else {
		// convention - if user ID is zero, we're displaying create form
		view.AddKey("publication", model.Publication{Title: "Temporary test"})
		view.AddKey("pageTitle", "Create publication")
	}
	view.Template("publications/form.html.got")
	return view, nil
}

// CreateOrUpdatePublication creates a publication in the database - form is "multipart/form-data" for both create and update
func CreateOrUpdatePublication(server http.IServer, pub *model.Publication) (*views.Renderer, error) {
	switch pub.ID {
	case 0:
		if len(pub.Files) != 1 {
			return nil, http.Problem{Detail: "Please upload only one file", Status: http.StatusBadRequest}
		}
		for _, file := range pub.Files {
			server.LogInfo("File : %s", file)
			// get the path to the master file
			if _, err := os.Stat(file); err != nil {
				// the master file does not exist
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
			}

			// encrypt the EPUB File and send the content to the LCP server
			err := encryptEPUBSendToLCP(file, http.Slugify(pub.Title), server)
			if err != nil {
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}

			err = server.Store().Publication().Add(pub)
			if err != nil {
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
			}
		}
	default:
		// searching for updated entity
		if existingPublication, err := server.Store().Publication().Get(pub.ID); err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		} else {
			// performing update
			if err = server.Store().Publication().Update(&model.Publication{ID: existingPublication.ID, UUID: existingPublication.UUID, Title: pub.Title, Status: pub.Status}); err != nil {
				//update failed!
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
	}
	return nil, http.Problem{Detail: "/publications", Status: http.StatusRedirect}
}

// CheckPublicationByTitle check if a publication with this title exist
func CheckPublicationByTitle(server http.IServer, param ParamTitle) (*views.Renderer, error) {

	server.LogInfo("Check publication stored with name %q", param.Title)

	if _, err := server.Store().Publication().CheckByTitle(param.Title); err == nil {
		//enc := json.NewEncoder(resp)
		//if err = enc.Encode(pub); err == nil {
		// send json of correctly encoded user info
		//	resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
		//	resp.WriteHeader(http.StatusOK)
		//	return nil, nil
		//}
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	} else {
		switch err {
		case gorm.ErrRecordNotFound:

			server.LogInfo("No publication stored with name %q", param.Title)
			//	server.Error(w, r, s.DefaultSrvLang(), common.Problem{Detail: err.Error(),Status: http.StatusNotFound)

		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

		}
	}
	return nil, nil
}

// DeletePublication removes a publication in the database
func DeletePublication(server http.IServer, param ParamId) (*views.Renderer, error) {
	ids := strings.Split(param.Id, ",")
	for _, id := range ids {
		server.LogInfo("Delete %s", id)
	}
	return &views.Renderer{}, http.Problem{Status: http.StatusOK}
	id, err := strconv.Atoi(param.Id)
	publication, err := server.Store().Publication().Get(int64(id))
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
	}

	// delete the epub file from the master repository
	inputPath := path.Join(server.Config().LutServer.MasterRepository, publication.Title+".epub")

	if _, err := os.Stat(inputPath); err == nil {
		err = os.Remove(inputPath)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		}
	}

	if err = server.Store().Publication().Delete(int64(id)); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	return nil, nil
}

// encryptEPUB encrypts an EPUB File and sends the content to the LCP server
func encryptEPUBSendToLCP(inputPath string, contentDisposition string, server http.IServer) error {

	// generate a new uuid; this will be the content id in the lcp server
	uid, errU := model.NewUUID()
	if errU != nil {
		return errU
	}
	contentUUID := uid.String()

	// create a temp file in the frontend "encrypted repository"
	outputFilename := contentUUID + ".tmp"
	outputPath := path.Join(server.Config().LutServer.EncryptedRepository, outputFilename)
	defer func() {
		// remove the temporary file in the "encrypted repository"
		err := os.Remove(outputPath)
		if err != nil {
			server.LogError("Error removing trash : %v", err)
		}
	}()
	// encrypt the master file found at inputPath, write in the temp file, in the "encrypted repository"
	encryptedEpub, err := pack.CreateEncryptedEpub(inputPath, outputPath)

	if err != nil {
		// unable to encrypt the master file
		if _, err := os.Stat(inputPath); err == nil {
			os.Remove(inputPath)
		}
		return err
	}

	// prepare the request for import to the lcp server
	lcpPublication := http.LcpPublication{
		ContentId:          contentUUID,
		ContentKey:         encryptedEpub.EncryptionKey,
		Output:             outputPath,
		ContentDisposition: &contentDisposition,
		Checksum:           &encryptedEpub.Checksum,
		Size:               &encryptedEpub.Size,
	}

	// json encode the payload
	jsonBody, err := json.Marshal(lcpPublication)
	if err != nil {
		return err
	}
	// send the content to the LCP server
	lcpURL := server.Config().LcpServer.PublicBaseUrl + "/contents/" + contentUUID

	req, err := http.NewRequest("PUT", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	// authenticate
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(server.Config().LcpUpdateAuth.Username, server.Config().LcpUpdateAuth.Password)
	}
	// FormToFields the payload type
	req.Header.Add(http.HdrContentType, http.ContentTypeLcpJson)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		server.LogError("Error PUT on LCP Server : %v", err)
		return err
	}

	// we have a body, defering close
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		// error on creation
		server.LogError("Bad PUT on LCP Server. Http status ", resp.StatusCode)
		return err
	}

	// reading body
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		server.LogError("Error PUT on LCP Server : reading body error : %v", err)
		return err
	}

	//log.Printf("Lsd Server on compliancetest response : %v [http-status:%d]", body, resp.StatusCode)

	return nil
}
