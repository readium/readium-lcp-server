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
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"github.com/adrg/errors"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"net"
)

// GetFilteredLicenses searches licenses activated by more than n devices
//
func GetFilteredLicenses(server http.IServer, param ParamPagination) (*views.Renderer, error) {
	view := &views.Renderer{}
	if param.Filter == "" {
		param.Filter = "1"
	} else {
		view.AddKey("filter", param.Filter)
	}
	noOfLicenses, err := server.Store().License().CountFiltered(param.Filter)
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfLicenses)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	licenses, err := server.Store().License().GetFiltered(param.Filter, perPage, page)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	if (page+1)*perPage < noOfLicenses {
		view.AddKey("hasNextPage", true)
	}
	view.AddKey("licenses", licenses)
	view.AddKey("pageTitle", "Licenses admin")
	view.AddKey("total", noOfLicenses)
	view.AddKey("currentPage", page+1)
	view.AddKey("perPage", perPage)
	view.Template("admin/index.html.got")
	return view, nil
}

// GetLicense gets an existing license by its id (passed as a section of the URL).
// It generates a partial license from the purchase info,
// fetches the license from the lcp server and returns it to the caller.
//
func GetLicense(server http.IServer, param ParamId) ([]byte, error) {
	purchase, err := server.Store().Purchase().GetByLicenseID(param.Id)
	// get the license id in the URL
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
	server.LogInfo("Downloading license : %q", param.Id)
	result, err := readLicenseFromLCP(purchase, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	err = server.Store().Purchase().MarkAsDelivered(param.Id)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Add(http.HdrContentType, http.ContentTypeLcpJson)
	nonErr.HttpHeaders.Set(http.HdrContentDisposition, "attachment; filename=\"license.lcpl\"")
	return result, nonErr
}

func CancelLicense(server http.IServer, param ParamId) http.Problem {
	server.LogInfo("Cancelling %q", param.Id)
	result, err := changeStatusToLSD(server, param.Id, "CANCEL")
	if err != nil {
		return http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	err = server.Store().License().BulkAddOrUpdate(model.LicensesStatusCollection{result})
	if err != nil {
		return http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	return http.Problem{Status: http.StatusRedirect}
}

func RevokeLicense(server http.IServer, param ParamId) http.Problem {
	server.LogInfo("Revoking %q", param.Id)
	result, err := changeStatusToLSD(server, param.Id, "REVOKE")
	if err != nil {
		return http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	err = server.Store().License().BulkAddOrUpdate(model.LicensesStatusCollection{result})
	if err != nil {
		return http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	return http.Problem{Status: http.StatusOK}
}

func changeStatusToLSD(server http.IServer, id, command string) (*model.LicenseStatus, error) {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		server.LogError("Error contacting LcpServer : %v", err)
		return nil, fmt.Errorf("LCP Server probably not running : %v", err)
	}
	defer conn.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString(command + "\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return nil, err
	}

	payload := http.AuthorizationAndLicense{
		License: &model.License{
			Id: id,
		},
		User:     server.Config().LcpUpdateAuth.Username,
		Password: server.Config().LcpUpdateAuth.Password,
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(payload)
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return nil, err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return nil, err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return nil, err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var result model.LicenseStatus
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&result)
		if err != nil {
			server.LogError("Error decoding GOB license : %v", err)
		} else {
			return &result, nil
		}
	} else if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return nil, fmt.Errorf(responseErr.Err)
	}
	return nil, err
}

func readLicenseFromLCP(fromPurchase *model.Purchase, server http.IServer) ([]byte, error) {
	if server.Config().LcpUpdateAuth.Username == "" {
		return nil, fmt.Errorf("Username is empty : can't connect to LCP.")
	}
	var err error
	var userKeyValue []byte

	if fromPurchase.User.Password == "" {
		return nil, errors.New("User has invalid, empty password - it's probably imported.")
	} else {
		userKeyValue, err = hex.DecodeString(fromPurchase.User.Password)
		if err != nil {
			server.LogError("Missing User Password [%q] or hex.DecodeString error : %v", fromPurchase.User.Password, err)
			return nil, err
		}
	}

	if fromPurchase.User.Hint == "" {
		return nil, errors.New("User has invalid, empty hint - it's probably imported.")
	}

	license := &model.License{
		Id:        fromPurchase.LicenseUUID.String,
		Provider:  server.Config().LutServer.ProviderUri,
		ContentId: fromPurchase.Publication.UUID,
		UserId:    fromPurchase.User.UUID,
		User: model.User{
			Email:     fromPurchase.User.Email,
			Name:      fromPurchase.User.Name,
			UUID:      fromPurchase.User.UUID,
			Encrypted: []string{"email", "name"},
		},
		Encryption: model.LicenseEncryption{
			UserKey: model.LicenseUserKey{
				Key: model.Key{
					Algorithm: "http://www.w3.org/2001/04/xmlenc#sha256",
				},
				Hint:  fromPurchase.User.Hint,
				Value: string(userKeyValue),
			},
		},
		Rights: &model.LicenseUserRights{
			Copy:  &model.NullInt{NullInt64: sql.NullInt64{Int64: server.Config().LutServer.RightCopy, Valid: true}},
			Print: &model.NullInt{NullInt64: sql.NullInt64{Int64: server.Config().LutServer.RightPrint, Valid: true}},
		},
	}
	// prepare the payload for import to the lcp server
	payload := http.AuthorizationAndLicense{
		License:  license,
		User:     server.Config().LcpUpdateAuth.Username,
		Password: server.Config().LcpUpdateAuth.Password,
	}

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		server.LogError("Error contacting LcpServer : %v", err)
		return nil, fmt.Errorf("LCP Server probably not running : %v", err)
	}
	defer conn.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("GETLICENSE\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return nil, err
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(payload)
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return nil, err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return nil, err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return nil, err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		return bodyBytes, nil
	} else if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return nil, fmt.Errorf(responseErr.Err)
	}
	return nil, nil
}
