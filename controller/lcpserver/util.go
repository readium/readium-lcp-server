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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type (
	ParamName struct {
		Name string `var:"name"`
	}

	ParamContentId struct {
		ContentID string `var:"content_id"`
	}

	ParamContentIdAndPage struct {
		ContentID string `var:"content_id"`
		http.ParamPagination
	}

	ParamLicenseId struct {
		LicenseID string `var:"license_id"`
	}
)

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

// notifyLSDServer informs the License Status Server of the creation of a new license
// and saves the result of the http request in the DB (using *LicenseRepository)
//
func notifyLSDServer(payload *model.License, server http.IServer) {
	if server.Config().LsdServer.PublicBaseUrl == "" {
		// can't call : url is empty
		return
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		server.LogError("Error Notify LsdServer of new License (" + payload.Id + ") Marshaling error : " + err.Error())
	}

	req, err := http.NewRequest("PUT", server.Config().LsdServer.PublicBaseUrl+"/licenses", bytes.NewReader(jsonPayload))
	if err != nil {
		return
	}

	// set credentials on lsd request
	notifyAuth := server.Config().LsdNotifyAuth
	if notifyAuth.Username != "" {
		req.SetBasicAuth(notifyAuth.Username, notifyAuth.Password)
	}
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
		server.LogError("Error Notify LsdServer of new License (" + payload.Id + "):" + err.Error())
		err = server.Store().License().UpdateLsdStatus(payload.Id, -1)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		server.LogError("Error Notify LsdServer of new License (read body error : %v)", err)
	}
	_ = server.Store().License().UpdateLsdStatus(payload.Id, int32(resp.StatusCode))
	// message to the console
	server.LogInfo("Notify LsdServer of a new License with id %q http-status %d response %v", payload.Id, resp.StatusCode, body)

}

// build a license, common to get and generate license, get and generate licensed publication
//
func buildLicense(license *model.License, server http.IServer) error {

	// set the LCP profile
	// possible profiles are basic and 1.0
	if server.Config().Profile == "1.0" {
		license.Encryption.Profile = model.V1Profile
	} else {
		license.Encryption.Profile = model.BasicProfile
	}

	// get content info from the db
	content, err := server.Store().Content().Get(license.ContentId)
	if err != nil {
		server.LogError("No content with id %v %v", license.ContentId, err)
		return err
	}

	// set links
	err = license.SetLinks(content)
	if err != nil {
		return err
	}

	// setting type - so model won't depend on these constants
	for i := 0; i < len(license.Links); i++ {
		switch license.Links[i].Rel {
		// publication link
		case "publication":
			license.Links[i].Type = epub.ContentTypeEpub
			// status link
		case "status":
			license.Links[i].Type = http.ContentTypeLsdJson
		}

	}

	// encrypt the content key, user fieds, set the key check
	err = license.EncryptLicenseFields(content)
	if err != nil {
		return err
	}

	// sign the license
	err = license.SignLicense(server.Certificate())
	if err != nil {
		return err
	}
	return nil
}

// build a licensed publication, common to get and generate licensed publication
//
func buildLicencedPublication(license *model.License, server http.IServer) (*epub.Epub, error) {
	// get the epub content info from the bd
	epubFile, err := server.Storage().Get(license.ContentId)
	if err != nil {
		return nil, err
	}
	// get the epub content
	epubContent, err1 := epubFile.Contents()
	if err1 != nil {
		return nil, err1
	}
	var b bytes.Buffer
	// copy the epub content to a buffer
	io.Copy(&b, epubContent)
	// create a zip reader
	zr, err2 := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err2 != nil {
		return nil, err2
	}
	ep, err3 := epub.Read(zr)
	if err3 != nil {
		return nil, err3
	}
	// add the license to publication
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(license)
	// write the buffer in the zip, and suppress the trailing newline
	// FIXME: check that the newline is not present anymore
	// FIXME/ try to optimize with buf.ReadBytes(byte('\n')) instead of creating a new buffer.
	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add(epub.LicenseFile, &buf2, uint64(buf2.Len()))
	return &ep, err
}

func RegisterRoutes(muxer *mux.Router, server http.IServer) {
	muxer.NotFoundHandler = server.NotFoundHandler() // handle all other requests 404
	// methods related to EPUB encrypted content

	contentRoutesPathPrefix := "/contents"
	contentRoutes := muxer.PathPrefix(contentRoutesPathPrefix).Subrouter().StrictSlash(false)

	server.HandleFunc(muxer, contentRoutesPathPrefix, ListContents, false).Methods("GET")

	// get encrypted content by content id (a uuid)
	server.HandleFunc(contentRoutes, "/{content_id}", GetContent, false).Methods("GET")
	// get all licenses associated with a given content
	server.HandleFunc(contentRoutes, "/{content_id}/licenses", ListLicensesForContent, true).Methods("GET")

	if !server.Config().LcpServer.ReadOnly {
		server.HandleFunc(contentRoutes, "/{name}", StoreContent, true).Methods("POST")
		// put content to the storage
		server.HandleFunc(contentRoutes, "/{content_id}", AddContent, true).Methods("PUT")
		// generate a license for given content
		server.HandleFunc(contentRoutes, "/{content_id}/license", GenerateLicense, true).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		server.HandleFunc(contentRoutes, "/{content_id}/licenses", GenerateLicense, true).Methods("POST")
		// generate a licensed publication
		server.HandleFunc(contentRoutes, "/{content_id}/publication", GenerateLicensedPublication, true).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		server.HandleFunc(contentRoutes, "/{content_id}/publications", GenerateLicensedPublication, true).Methods("POST")
	}

	// methods related to licenses

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	server.HandleFunc(muxer, licenseRoutesPathPrefix, ListLicenses, true).Methods("GET")
	// get a license
	server.HandleFunc(licenseRoutes, "/{license_id}", GetLicense, true).Methods("GET")
	server.HandleFunc(licenseRoutes, "/{license_id}", GetLicense, true).Methods("POST")
	// get a licensed publication via a license id
	server.HandleFunc(licenseRoutes, "/{license_id}/publication", GetLicensedPublication, true).Methods("POST")
	if !server.Config().LcpServer.ReadOnly {
		// update a license
		server.HandleFunc(licenseRoutes, "/{license_id}", UpdateLicense, true).Methods("PATCH")
	}
}
