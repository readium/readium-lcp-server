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
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
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
// if ID is zero, we're displaying create form
func GetPublication(server http.IServer, param ParamId) (*views.Renderer, error) {
	view := &views.Renderer{}
	var publication *model.Publication
	if param.Id != "0" {
		id, err := strconv.Atoi(param.Id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
		}
		publication, err = server.Store().Publication().Get(int64(id))
		if err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}

			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
		view.AddKey("pageTitle", "Edit publication")
	} else {
		fileListing, err := ioutil.ReadDir(server.Config().LutServer.MasterRepository)
		if err == nil {
			var files []RepositoryFile
			for _, file := range fileListing {
				fileExt := filepath.Ext(file.Name())
				if fileExt == ".epub" {
					files = append(files, RepositoryFile{Name: file.Name()})
				}
			}
			view.AddKey("existingFiles", files)
		} else {
			server.LogError("Error reading repository files : %v", err)
		}

		publication = &model.Publication{}
		view.AddKey("pageTitle", "Create publication")
	}
	view.AddKey("publication", publication)
	view.Template("publications/form.html.got")
	return view, nil
}

// CreateOrUpdatePublication creates a publication in the database - form is "multipart/form-data" for both create and update
func CreateOrUpdatePublication(server http.IServer, pub *model.Publication) (*views.Renderer, error) {
	switch pub.ID {
	case 0:
		if pub.RepoFile == "" && len(pub.Files) != 1 {
			return nil, http.Problem{Detail: "Please upload only one file", Status: http.StatusBadRequest}
		}
		// first we're generating an UUID, before sending to LCP (so we have the same reference)
		uid, errU := model.NewUUID()
		if errU != nil {
			return nil, http.Problem{Detail: "Failed to generate UUID " + errU.Error(), Status: http.StatusInternalServerError}
		}
		pub.UUID = uid.String()

		// case : uploaded files
		for _, file := range pub.Files {
			//server.LogInfo("File : %s", file)
			// get the path to the master file
			if _, err := os.Stat(file); err != nil {
				// the master file does not exist
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
			}

			// encrypt the EPUB File and send the content to the LCP server
			err := encryptEPUBSendToLCP(file, pub.UUID, http.Slugify(pub.Title), server)
			if err != nil {
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}

			err = server.Store().Publication().Add(pub)
			if err != nil {
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
			}
		}

		// case : chosen files
		if pub.RepoFile != "" {
			err := encryptEPUBSendToLCP(server.Config().LutServer.MasterRepository+"/"+pub.RepoFile, pub.UUID, http.Slugify(pub.Title), server)
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
			if existingPublication.Title == pub.Title {
				// nothing changed : the user pressed save for nothing
				return nil, http.Problem{Detail: "/publications", Status: http.StatusRedirect}
			}
			payload := &model.Publication{
				ID:    existingPublication.ID,
				UUID:  existingPublication.UUID,
				Title: pub.Title,
			}
			err = sendTitleUpdateToLCP(existingPublication.UUID, http.Slugify(pub.Title), server)
			if err != nil {
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
			// performing update
			if err = server.Store().Publication().Update(payload); err != nil {
				//update failed!
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
	}
	return nil, http.Problem{Detail: "/publications", Status: http.StatusRedirect}
}

// DeletePublication removes a publication in the database and LCP
func DeletePublication(server http.IServer, param ParamId) (*views.Renderer, error) {
	ids := strings.Split(param.Id, ",")
	var pubIds []int64
	var deletedTitles []string
	for _, sid := range ids {
		id, err := strconv.Atoi(sid)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
		}
		publication, err := server.Store().Publication().Get(int64(id))
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		}

		// tell LCP to delete content and licenses
		err = deleteContentFromLCP(publication.UUID, server)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		// everything ok, adding to bulk delete
		deletedTitles = append(deletedTitles, publication.Title)
		pubIds = append(pubIds, int64(id))
	}

	if err := server.Store().Publication().BulkDelete(pubIds); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}

	// attempt to delete the epubs file from the master repository
	for _, title := range deletedTitles {
		inputPath := path.Join(server.Config().LutServer.MasterRepository, title+".epub")
		if _, err := os.Stat(inputPath); err == nil {
			err = os.Remove(inputPath)
			if err != nil {
				// silent fail
				server.LogError("Error removing epub %s from %s : %v", title, server.Config().LutServer.MasterRepository, err)
			}
		} else {
			server.LogError("Error finding epub %s from %s : %v", title, server.Config().LutServer.MasterRepository, err)
		}
	}

	return &views.Renderer{}, http.Problem{Status: http.StatusOK}
}

func CheckPublicationTitleExists(server http.IServer, param ParamTitleAndId) ([]byte, error) {
	id, err := strconv.Atoi(param.Id)
	if err != nil {
		// id is not a number
		return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
	}
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	publication, err := server.Store().Publication().CheckByTitle(param.Title)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// returns StatusOK (frontend proceed)
			return nil, nonErr
		}
		server.LogError("Error checking for publication with tile %q : %v", param.Title, err)
		// returns StatusInternalServerError (frontend uncertain)
		nonErr.Status = http.StatusInternalServerError
		return nil, nonErr
	}
	if id == int(publication.ID) {
		// returns StatusOK (is the very same record - allow it to save)
		return nil, nonErr
	}
	server.LogInfo("Found same title %d?=%d %s", id, publication.ID, publication.Title)
	// returns StatusBadRequest (frontend deny creation)
	nonErr.Status = http.StatusBadRequest
	return nil, nonErr
}

// encryptEPUB encrypts an EPUB File and sends the content to the LCP server
func encryptEPUBSendToLCP(inputPath, contentUUID, contentDisposition string, server http.IServer) error {
	if server.Config().LcpUpdateAuth.Username == "" {
		return fmt.Errorf("Username is empty : can't connect to LCP.")
	}

	// create a temp file in the frontend "encrypted repository"
	outputFilename := contentUUID + ".tmp"
	outputPath := path.Join(server.Config().LutServer.EncryptedRepository, outputFilename)
	// prepare cleanup
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

	// prepare the payload for import to the lcp server
	lcpPublication := http.AuthorizationAndLcpPublication{
		ContentId:          contentUUID,
		ContentKey:         encryptedEpub.EncryptionKey,
		Output:             outputPath,
		ContentDisposition: &contentDisposition,
		Checksum:           &encryptedEpub.Checksum,
		Size:               &encryptedEpub.Size,
		User:               server.Config().LcpUpdateAuth.Username,
		Password:           server.Config().LcpUpdateAuth.Password,
	}

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		server.LogError("Error Notify LcpServer : %v", err)
		return fmt.Errorf("LCP Server probably not running : %v", err)
	}
	defer conn.Close()
	server.LogInfo("Notifying LCP (creating content).")
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("CREATECONTENT\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return err
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(lcpPublication)
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var content model.Content
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&content)
		if err != nil {
			server.LogError("Error decoding GOB content : %v", err)
		} else {
			server.LogInfo("Model content : %#v", content)
		}
	} else if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return fmt.Errorf(responseErr.Err)
	}
	return nil
}

func sendTitleUpdateToLCP(contentUUID, contentDisposition string, server http.IServer) error {
	if server.Config().LcpUpdateAuth.Username == "" {
		return fmt.Errorf("Username is empty : can't connect to LCP.")
	}

	// prepare the payload for import to the lcp server
	lcpPublication := http.AuthorizationAndLcpPublication{
		ContentId:          contentUUID,
		ContentDisposition: &contentDisposition,
		User:               server.Config().LcpUpdateAuth.Username,
		Password:           server.Config().LcpUpdateAuth.Password,
	}

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		server.LogError("Error Notify LcpServer : %v", err)
		return fmt.Errorf("LCP Server probably not running : %v", err)
	}
	defer conn.Close()
	server.LogInfo("Notifying LCP (updating content title).")
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("UPDATECONTENT\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return err
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(lcpPublication)
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var content model.Content
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&content)
		if err != nil {
			server.LogError("Error decoding GOB content : %v", err)
		} else {
			server.LogInfo("Model content : %#v", content)
		}
	} else if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return fmt.Errorf(responseErr.Err)
	}
	return nil
}

func deleteContentFromLCP(contentUUID string, server http.IServer) error {
	if server.Config().LcpUpdateAuth.Username == "" {
		return fmt.Errorf("Username is empty : can't connect to LCP.")
	}

	// prepare the payload for import to the lcp server
	lcpPublication := http.AuthorizationAndLcpPublication{
		ContentId: contentUUID,
		User:      server.Config().LcpUpdateAuth.Username,
		Password:  server.Config().LcpUpdateAuth.Password,
	}

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		server.LogError("Error Notify LcpServer : %v", err)
		return fmt.Errorf("LCP Server probably not running : %v", err)
	}
	defer conn.Close()
	server.LogInfo("Notifying LCP (updating content title).")
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("DELETECONTENT\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return err
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(lcpPublication)
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		// nothing to do
	} else if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return fmt.Errorf(responseErr.Err)
	}
	return nil
}
