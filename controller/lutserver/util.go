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
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/store"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

type (
	// Pagination used to paginate listing
	Pagination struct {
		Page    int
		PerPage int
	}
	// EncryptedEpub Encrypted epub

	EncryptedEpub struct {
		Path          string
		EncryptionKey []byte
		Size          int64
		Checksum      string
	}

	// Contains all repository definitions
	RepositoryManager struct {
		MasterRepositoryPath    string
		EncryptedRepositoryPath string
	}

	// WebRepository interface for repository db interaction
	WebRepository interface {
		GetMasterFile(name string) (RepositoryFile, error)
		GetMasterFiles() func() (RepositoryFile, error)
	}

	// RepositoryFile struct defines a file stored in a repository
	RepositoryFile struct {
		Name string `json:"name"`
		Path string
	}
)

var ErrNotFound = errors.New("License not found")
var repoInited bool
var repoManager RepositoryManager

// Returns a specific repository file
func (repManager RepositoryManager) GetMasterFile(name string) (RepositoryFile, error) {
	var filePath = path.Join(repManager.MasterRepositoryPath, name)

	if _, err := os.Stat(filePath); err == nil {
		// File exists
		var repFile RepositoryFile
		repFile.Name = name
		repFile.Path = filePath
		return repFile, err
	}

	return RepositoryFile{}, ErrNotFound
}

// Returns all repository files
func (repManager RepositoryManager) GetMasterFiles() func() (RepositoryFile, error) {
	files, err := ioutil.ReadDir(repManager.MasterRepositoryPath)
	var fileIndex int

	if err != nil {
		return func() (RepositoryFile, error) { return RepositoryFile{}, err }
	}

	return func() (RepositoryFile, error) {
		var result RepositoryFile

		// Filter on epub
		for fileIndex < len(files) {
			file := files[fileIndex]
			fileExt := filepath.Ext(file.Name())
			fileIndex++

			if fileExt == ".epub" {
				result.Name = file.Name()
				return result, err
			}
		}

		return result, ErrNotFound
	}
}

// GetRepositoryMasterFiles returns a list of repository masterfiles
func GetRepositoryMasterFiles(w http.ResponseWriter, r *http.Request, s api.IServer) {
	var err error

	files := make([]RepositoryFile, 0)
	if !repoInited {
		repoManager = RepositoryManager{MasterRepositoryPath: s.Config().FrontendServer.MasterRepository, EncryptedRepositoryPath: s.Config().FrontendServer.EncryptedRepository}
	}
	fn := repoManager.GetMasterFiles()

	for it, err := fn(); err == nil; it, err = fn() {
		files = append(files, it)
	}

	w.Header().Set(api.HdrContentType, api.ContentTypeJson)

	enc := json.NewEncoder(w)
	err = enc.Encode(files)
	if err != nil {
		api.Error(w, r, s.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// CreateEncryptedEpub Encrypt input file to output file
func CreateEncryptedEpub(inputPath string, outputPath string) (EncryptedEpub, error) {
	if _, err := os.Stat(inputPath); err != nil {
		return EncryptedEpub{}, errors.New("Input file does not exists")
	}

	// Read file
	buf, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to read input file")
	}

	// Read the epub content from the zipped buffer
	zipReader, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return EncryptedEpub{}, errors.New("Invalid zip (epub) file")
	}

	epubContent, err := epub.Read(zipReader)
	if err != nil {
		return EncryptedEpub{}, errors.New("Invalid epub content")
	}

	// Create output file
	output, err := os.Create(outputPath)
	if err != nil {
		wd, err := os.Getwd()
		if err != nil {
			panic("Cannot read working dir.")
		}
		return EncryptedEpub{}, fmt.Errorf("Unable to create output file : %s (%s)", outputPath, wd)
	}

	// Pack / encrypt the epub content, fill the output file
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	_, encryptionKey, err := pack.Do(encrypter, epubContent, output)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to encrypt file")
	}

	stats, err := output.Stat()
	if err != nil || (stats.Size() <= 0) {
		return EncryptedEpub{}, errors.New("Unable to output file")
	}

	hasher := sha256.New()
	s, err := ioutil.ReadFile(outputPath)
	_, err = hasher.Write(s)
	if err != nil {
		return EncryptedEpub{}, errors.New("Unable to build checksum")
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	output.Close()
	return EncryptedEpub{outputPath, encryptionKey, stats.Size(), checksum}, nil
}

// EncryptEPUB encrypts an EPUB File and sends the content to the LCP server
func EncryptEPUB(inputPath string, contentDisposition string, server api.IServer) error {

	// generate a new uuid; this will be the content id in the lcp server
	uid, errU := uuid.NewV4()
	if errU != nil {
		return errU
	}
	contentUUID := uid.String()

	// create a temp file in the frontend "encrypted repository"
	outputFilename := contentUUID + ".tmp"
	outputPath := path.Join(server.Config().FrontendServer.EncryptedRepository, outputFilename)
	defer func() {
		// remove the temporary file in the "encrypted repository"
		err := os.Remove(outputPath)
		if err != nil {
			server.LogError("Error removing trash : %v", err)
		}
	}()
	// encrypt the master file found at inputPath, write in the temp file, in the "encrypted repository"
	encryptedEpub, err := CreateEncryptedEpub(inputPath, outputPath)

	if err != nil {
		// unable to encrypt the master file
		if _, err := os.Stat(inputPath); err == nil {
			os.Remove(inputPath)
		}
		return err
	}

	// prepare the request for import to the lcp server
	lcpPublication := api.LcpPublication{
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
	lcpServerConfig := server.Config().LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/contents/" + contentUUID
	//log.Println("PUT " + lcpURL)

	req, err := http.NewRequest("PUT", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	// authenticate
	lcpUpdateAuth := server.Config().LcpUpdateAuth
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// set the payload type
	req.Header.Add(api.HdrContentType, api.ContentTypeLcpJson)

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

// ExtractPaginationFromRequest extract from http.Request pagination information
func ExtractPaginationFromRequest(req *http.Request) (Pagination, error) {
	var err error
	var page int64    // default: page 1
	var perPage int64 // default: 30 items per page
	pagination := Pagination{}

	if req.FormValue("page") != "" {
		page, err = strconv.ParseInt((req).FormValue("page"), 10, 32)
		if err != nil {
			return pagination, err
		}
	} else {
		page = 1
	}

	if req.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((req).FormValue("per_page"), 10, 32)
		if err != nil {
			return pagination, err
		}
	} else {
		perPage = 30
	}

	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}

	if page < 0 {
		return pagination, err
	}

	pagination.Page = int(page)
	pagination.PerPage = int(perPage)
	return pagination, err
}

// PrepareListHeaderResponse set several http headers
// sets previous and next link headers
func PrepareListHeaderResponse(resourceCount int, resourceLink string, pagination Pagination, resp http.ResponseWriter) {
	if resourceCount > 0 {
		nextPage := strconv.Itoa(int(pagination.Page) + 1)
		resp.Header().Set("Link", "<"+resourceLink+"?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if pagination.Page > 1 {
		previousPage := strconv.Itoa(int(pagination.Page) - 1)
		resp.Header().Set("Link", "<"+resourceLink+"/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	resp.Header().Set(api.HdrContentType, api.ContentTypeJson)
}

func generateOrGetLicense(purchase *store.Purchase, server api.IServer) (*store.License, error) {
	// create a partial license
	partialLicense := store.License{}

	// set the mandatory provider URI
	if server.Config().FrontendServer.ProviderUri == "" {
		return nil, errors.New("Mandatory provider URI missing in the configuration")
	}
	partialLicense.Provider = server.Config().FrontendServer.ProviderUri

	// get user info from the purchase info
	encryptedAttrs := []string{"email", "name"}
	partialLicense.User.Email = purchase.User.Email
	partialLicense.User.Name = purchase.User.Name
	partialLicense.User.UUID = purchase.User.UUID
	partialLicense.User.Encrypted = encryptedAttrs

	// get the hashed passphrase from the purchase
	userKeyValue, err := hex.DecodeString(purchase.User.Password)

	if err != nil {
		return nil, err
	}

	userKey := store.LicenseUserKey{}
	userKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	userKey.Hint = purchase.User.Hint
	userKey.Value = string(userKeyValue)
	partialLicense.Encryption.UserKey = userKey

	// In case of a creation of license, add the user rights
	if purchase.LicenseUUID == nil {
		// in case of undefined conf values for copy and print rights,
		// these rights will be set to zero
		copyVal := server.Config().FrontendServer.RightCopy
		printVal := server.Config().FrontendServer.RightPrint
		userRights := store.LicenseUserRights{
			Copy:  &store.NullInt{NullInt64: sql.NullInt64{Int64: copyVal, Valid: true}},
			Print: &store.NullInt{NullInt64: sql.NullInt64{Int64: printVal, Valid: true}},
		}

		// if this is a loan, include start and end dates from the purchase info
		if purchase.Type == store.LOAN {
			userRights.Start = purchase.StartDate
			userRights.End = purchase.EndDate
		}

		partialLicense.Rights = &userRights
	}

	// encode in json
	jsonBody, err := json.Marshal(partialLicense)
	if err != nil {
		return nil, err
	}

	// get the url of the lcp server
	lcpServerConfig := server.Config().LcpServer
	var lcpURL string

	if purchase.LicenseUUID == nil || !purchase.LicenseUUID.Valid {
		// if the purchase contains no license id, generate a new license
		lcpURL = lcpServerConfig.PublicBaseUrl + "/contents/" + purchase.Publication.UUID + "/license"
	} else {
		// if the purchase contains a license id, fetch an existing license
		// note: this will not update the license rights
		lcpURL = lcpServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String
	}
	// message to the console
	//log.Println("POST " + lcpURL)

	// add the partial license to the POST request
	req, err := http.NewRequest("POST", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	lcpUpdateAuth := server.Config().LcpUpdateAuth
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// the body is a partial license in json format
	req.Header.Add("Content-Type", api.ContentTypeLcpJson)

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
		server.LogError("Error POST on LCP Server : %v", err)
		return nil, err
	}

	// we have a body, defering close
	defer resp.Body.Close()

	// if the status code from the request to the lcp server
	// is neither 201 Created or 200 ok, return an internal error
	if (purchase.LicenseUUID == nil && resp.StatusCode != 201) ||
		(purchase.LicenseUUID != nil && resp.StatusCode != 200) {
		return nil, errors.New("The License Server returned an error")
	}

	// decode the full license
	fullLicense := &store.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(fullLicense)

	if err != nil {
		return nil, errors.New("Unable to decode license")
	}

	// store the license id if it was not already set
	if purchase.LicenseUUID == nil {
		purchase.LicenseUUID = &store.NullString{NullString: sql.NullString{String: fullLicense.Id, Valid: true}}
		err = updatePurchase(purchase, server)
		if err != nil {
			return fullLicense, err
		}
	}

	return fullLicense, nil
}

// Update modifies a purchase on a renew or return request
// parameters: a Purchase structure withID,	LicenseUUID, StartDate,	EndDate, Status
// EndDate may be undefined (nil), in which case the lsd server will choose the renew period
//
func updatePurchase(purchase *store.Purchase, server api.IServer) error {
	// Get the original purchase from the db
	origPurchase, err := server.Store().Purchase().Get(purchase.ID)

	if err != nil {
		return fmt.Errorf("Error : reading purchase with id %d", purchase.ID)
	}
	if origPurchase.Status != store.StatusOk {
		return errors.New("Cannot update an invalid purchase")
	}
	if purchase.Status == store.StatusToBeRenewed ||
		purchase.Status == store.StatusToBeReturned {

		if purchase.LicenseUUID == nil {
			return errors.New("Cannot return or renew a purchase when no license has been delivered")
		}

		lsdServerConfig := server.Config().LsdServer
		lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String

		if purchase.Status == store.StatusToBeRenewed {
			lsdURL += "/renew"

			if purchase.EndDate != nil {
				lsdURL += "?end=" + purchase.EndDate.Time.Format(time.RFC3339)
			}

			// Next status if LSD raises no error
			purchase.Status = store.StatusOk
		} else if purchase.Status == store.StatusToBeReturned {
			lsdURL += "/return"

			// Next status if LSD raises no error
			purchase.Status = store.StatusOk
		}
		// message to the console
		//log.Println("PUT " + lsdURL)
		// prepare the request for renew or return to the license status server
		req, err := http.NewRequest("PUT", lsdURL, nil)
		if err != nil {
			return err
		}
		// set credentials
		lsdAuth := server.Config().LsdNotifyAuth
		if lsdAuth.Username != "" {
			req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
		}
		// call the lsd server

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

		defer resp.Body.Close()

		// get the new end date from the license server

		// FIXME: really needed? heavy...
		license, err := getPartialLicense(origPurchase, server)
		if err != nil {
			return err
		}
		purchase.EndDate = license.Rights.End
	} else {
		// status is not "to be renewed"
		purchase.Status = store.StatusOk
	}
	err = server.Store().Purchase().Update(purchase)
	if err != nil {
		return errors.New("Unable to update the license id")
	}
	return nil
}

func getPartialLicense(purchase *store.Purchase, server api.IServer) (*store.License, error) {

	if purchase.LicenseUUID == nil {
		return nil, errors.New("No license has been yet delivered")
	}

	lcpServerConfig := server.Config().LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String
	// message to the console
	//log.Println("GET " + lcpURL)
	// prepare the request
	req, err := http.NewRequest("GET", lcpURL, nil)
	if err != nil {
		return nil, err
	}
	// set credentials
	lcpUpdateAuth := server.Config().LcpUpdateAuth
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// send the request
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
		server.LogError("Error GET on LCP Server : %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	// the call must return 206 (partial content) because there is no input partial license
	if resp.StatusCode != 206 {
		// bad status code
		return nil, errors.New("The License Server returned an error")
	}
	// decode the license
	partialLicense := store.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&partialLicense)

	if err != nil {
		return nil, errors.New("Unable to decode the license")
	}

	return &partialLicense, nil
}
