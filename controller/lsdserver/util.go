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

package lsdserver

import (
	"strings"
	"time"

	"io/ioutil"

	"bufio"
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/i18n"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"net"
)

type (
	ParamKey struct {
		Key string `var:"key"`
	}

	ParamKeyAndDevice struct {
		Key        string `var:"key"`
		DeviceID   string `form:"id"`
		DeviceName string `form:"name"`
		End        string `form:"end"`
	}

	ParamDevicesAndPage struct {
		Devices string `form:"devices"`
		Page    string `form:"page"`
		PerPage string `form:"per_page"`
	}

	ParamLog struct {
		Stage  string `form:"test_stage"`
		Number string `form:"test_number"`
		Result string `form:"test_result"`
	}

	Headers struct {
		UserAgent      string // attention : this doesn't require `hdr:"User-Agent"`
		AcceptLanguage string `hdr:"Accept-Language"`
	}
)

// getEvents gets the events from database for the license status
//
func getEvents(payload *model.LicenseStatus, repo model.TransactionRepository) error {
	var err error
	payload.Events, err = repo.GetByLicenseStatusId(payload.Id)
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

// makeLinks creates and adds links to the license status
//
func makeLinks(payload *model.LicenseStatus, lsdConfig http.LsdServerInfo, lcpConfig http.ServerInfo, licStatus http.LicenseStatus) {
	licenseHasRightsEnd := payload.CurrentEndLicense.Valid && !(payload.CurrentEndLicense.Time).IsZero()
	returnAvailable := licStatus.Return && licenseHasRightsEnd
	renewAvailable := licStatus.Renew && licenseHasRightsEnd

	links := make(model.LicenseLinksCollection, 0, 0)

	if lsdConfig.LicenseLinkUrl != "" {
		licenseLinkURLReal := strings.Replace(lsdConfig.LicenseLinkUrl, "{license_id}", payload.LicenseRef, -1)
		link := &model.LicenseLink{
			Href:      licenseLinkURLReal,
			Rel:       "license",
			Type:      http.ContentTypeLcpJson,
			Templated: false,
		}
		links = append(links, link)
	} else {
		link := &model.LicenseLink{
			Href:      lcpConfig.PublicBaseUrl + "/licenses/" + payload.LicenseRef,
			Rel:       "license",
			Type:      http.ContentTypeLcpJson,
			Templated: false,
		}
		links = append(links, link)
	}

	if licStatus.Register {
		link := &model.LicenseLink{
			Href:      lsdConfig.PublicBaseUrl + "/licenses/" + payload.LicenseRef + "/register{?id,name}",
			Rel:       "register",
			Type:      http.ContentTypeLsdJson,
			Templated: true,
		}
		links = append(links, link)
	}

	if returnAvailable {
		link := &model.LicenseLink{
			Href:      lsdConfig.PublicBaseUrl + "/licenses/" + payload.LicenseRef + "/return{?id,name}",
			Rel:       "return",
			Type:      http.ContentTypeLsdJson,
			Templated: true,
		}
		links = append(links, link)
	}

	if renewAvailable {
		link := &model.LicenseLink{
			Href:      lsdConfig.PublicBaseUrl + "/licenses/" + payload.LicenseRef + "/renew{?end,id,name}",
			Rel:       "renew",
			Type:      http.ContentTypeLsdJson,
			Templated: true,
		}
		links = append(links, link)
	}

	payload.Links = links
}

// makeEvent creates an event and fill it
//
func makeEvent(status model.Status, deviceName string, deviceID string, licenseStatusFk int64) *model.TransactionEvent {
	return &model.TransactionEvent{
		DeviceId:        deviceID,
		DeviceName:      deviceName,
		Timestamp:       time.Now().UTC().Truncate(time.Second),
		Type:            status,
		LicenseStatusFk: licenseStatusFk,
	}
}

// notifyLCPServer updates a license by calling the License Server
// called from return, renew and cancel/revoke actions
//
func notifyLCPServer(timeEnd time.Time, licenseID string, server http.IServer) (int, error) {
	// create a minimum license object, limited to the license id plus rights
	// FIXME: remove the id (here and in the lcpserver license.go)
	minLicense := &model.License{Id: licenseID, Rights: &model.LicenseUserRights{End: &model.NullTime{Valid: true, Time: timeEnd}}}

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		server.LogError("Error Notify LcpServer : %v", err)
		return http.StatusInternalServerError, err
	}
	defer conn.Close()
	server.LogInfo("Notifying LCP about license update")
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	_, err = rw.WriteString("UPDATELICENSE\n")
	if err != nil {
		server.LogError("Could not write : %v", err)
		return http.StatusInternalServerError, err
	}

	notifyAuth := server.Config().LcpUpdateAuth
	if notifyAuth.Username == "" {
		server.LogError("Username is empty : can't connect to LCP")
		return http.StatusInternalServerError, fmt.Errorf("Username is empty : can't connect to LCP.")
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(http.AuthorizationAndLicense{User: notifyAuth.Username, Password: notifyAuth.Password, License: minLicense})
	if err != nil {
		server.LogError("Encode failed for struct: %v", err)
		return http.StatusInternalServerError, err
	}

	err = rw.Flush()
	if err != nil {
		server.LogError("Flush failed : %v", err)
		return http.StatusInternalServerError, err
	}
	// Read the reply.
	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		server.LogError("Error reading LCP reply : %v", err)
		return http.StatusInternalServerError, err
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		server.LogError("Error decoding LCP GOB : %v", err)
		return http.StatusInternalServerError, err
	}
	if responseErr.Err != "" {
		server.LogError("LCP GOB Error : %v", responseErr)
		return http.StatusInternalServerError, fmt.Errorf(responseErr.Err)
	}

	return http.StatusOK, nil
}

// fillLicenseStatus fills the localized 'message' field, the 'links' and 'event' objects in the license status
//
func fillLicenseStatus(payload *model.LicenseStatus, hdr Headers, s http.IServer) error {
	// add the localized message
	acceptLanguages := hdr.AcceptLanguage
	license := ""
	i18n.LocalizeMessage(s.Config().Localization.DefaultLanguage, acceptLanguages, &license, payload.Status.String())
	// add the links
	makeLinks(payload, s.Config().LsdServer, s.Config().LcpServer, s.Config().LicenseStatus)
	// add the events
	err := getEvents(payload, s.Store().Transaction())

	return err
}

// makeLicenseStatus sets fields of license status according to the config file
// and creates needed inner objects of license status
//
func makeLicenseStatus(license *model.License, registerAvailable bool, rentingDays int) *model.LicenseStatus {
	result := model.LicenseStatus{
		LicenseRef: license.Id,
	}

	if license.Rights == nil || license.Rights.Start == nil || license.Rights.End == nil {
		// The publication was purchased (not a loan), so we do not set LSD.PotentialRights.End
		result.CurrentEndLicense = &model.NullTime{
			Valid: false,
		}
	} else {
		// license.Rights.End exists => this is a loan
		endFromLicense := license.Rights.End.Time.Add(0)
		result.CurrentEndLicense = &model.NullTime{Time: endFromLicense, Valid: true}
		if rentingDays > 0 {
			endFromConfig := license.Issued.Add(time.Hour * 24 * time.Duration(rentingDays))
			if endFromLicense.After(endFromConfig) {
				result.PotentialRightsEnd = &model.NullTime{Time: endFromLicense, Valid: true}
			} else {
				result.PotentialRightsEnd = &model.NullTime{Time: endFromConfig, Valid: true}
			}
		} else {
			result.PotentialRightsEnd = &model.NullTime{Time: endFromLicense, Valid: true}
		}
	}

	if registerAvailable {
		result.Status = model.StatusReady
	} else {
		result.Status = model.StatusActive
	}

	result.LicenseUpdated = license.Issued
	result.StatusUpdated = time.Now()
	result.DeviceCount = &model.NullInt{NullInt64: sql.NullInt64{Int64: 1, Valid: true}} // default is 1, so it can be found on filter
	return &result
}

func RegisterRoutes(muxer *mux.Router, server http.IServer) {
	muxer.NotFoundHandler = server.NotFoundHandler() //handle all other requests 404

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)
	server.HandleFunc(muxer, licenseRoutesPathPrefix, FilterLicenseStatuses, true).Methods("GET")
	server.HandleFunc(licenseRoutes, "/{key}/status", GetLicenseStatusDocument, false).Methods("GET") // Note : this is unsecured
	if server.Config().ComplianceMode {
		//server.LogInfo("Compliance mode is ON.")
		server.HandleFunc(muxer, "/compliancetest", AddLogToFile, false).Methods("POST") // Note :  this is unsecured
	}

	server.HandleFunc(licenseRoutes, "/{key}/registered", ListRegisteredDevices, true).Methods("GET")
	if !server.Config().LsdServer.ReadOnly {
		server.HandleFunc(licenseRoutes, "/{key}/register", RegisterDevice, false).Methods("POST") // Note : this is unsecured
		server.HandleFunc(licenseRoutes, "/{key}/return", LendingReturn, false).Methods("PUT")     // Note : this is unsecured
		server.HandleFunc(licenseRoutes, "/{key}/renew", LendingRenewal, false).Methods("PUT")     // Note : this is unsecured
		server.HandleFunc(licenseRoutes, "/{key}/status", LendingCancellation, true).Methods("PATCH")
		server.HandleFunc(licenseRoutes, "", CreateLicenseStatusDocument, true).Methods("PUT")
	}

	// Gob encoding server provider.
	endpoint := http.NewGobEndpoint(server.Logger())
	endpoint.AddHandleFunc("LICENSES", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndTimestamp
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : `" + err.Error() + "`. Username and password are missing?")
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}

		var result model.LicensesStatusCollection
		if payload.Sync.IsZero() {
			result, err = server.Store().LicenseStatus().ListAll()
			if err != nil {
				return fmt.Errorf("Error reading license statuses : " + err.Error())
			}
			server.LogInfo("Listing all licenses statuses (%d found).", len(result))
		} else {
			server.LogInfo("Listing licenses statuses older than %v", payload.Sync)
			result, err = server.Store().LicenseStatus().ListLatest(payload.Sync)
			if err != nil {
				return fmt.Errorf("Error reading license statuses : " + err.Error())
			}
		}

		enc := gob.NewEncoder(rw)
		err = enc.Encode(result)
		if err != nil {
			return fmt.Errorf("Error encoding result : " + err.Error())
		}
		return nil
	})

	endpoint.AddHandleFunc("LICENSESDELETED", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndLicense
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : " + err.Error())
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}
		server.LogInfo("Received GOB License : %#v", payload.License)
		err = server.Store().LicenseStatus().DeleteByLicenseIds(payload.License.Id)
		if err != nil {
			server.LogError("Error deleting licenses for ids %q : %v", payload.License.Id, err)
		}
		return nil
	})

	endpoint.AddHandleFunc("UPDATELICENSESTATUS", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndLicense
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : " + err.Error())
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}
		server.LogInfo("Received GOB License : %#v", payload.License)
		ls := makeLicenseStatus(payload.License, server.Config().LicenseStatus.Register, server.Config().LicenseStatus.RentingDays)

		err = server.Store().LicenseStatus().Add(ls)
		if err != nil {
			return fmt.Errorf("Error saving license : " + err.Error())
		}

		return nil
	})

	endpoint.AddHandleFunc("RENEW", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndLicense
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : " + err.Error())
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}
		server.LogInfo("Received GOB License : %#v", payload.License)
		//licenseStatus := &model.LicenseStatus{}
		LendingRenewal(server, ParamKeyAndDevice{Key: payload.License.Id, DeviceID: "system", DeviceName: "system", End: payload.License.End.Time.Format(time.RFC3339)}, Headers{AcceptLanguage: "en_US"})
		return nil
	})

	endpoint.AddHandleFunc("CANCEL", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndLicense
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : " + err.Error())
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}
		//server.LogInfo("Received GOB License : %#v", payload.License)
		server.LogInfo("Cancelling license with id %q", payload.License.Id)
		licenseStatus := &model.LicenseStatus{Status: model.StatusCancelled}
		_, err = LendingCancellation(server, licenseStatus, ParamKey{payload.License.Id})
		if err != nil {
			return err
		}
		result, err := server.Store().LicenseStatus().GetByLicenseId(payload.License.Id)
		if err != nil {
			return err
		}
		enc := gob.NewEncoder(rw)
		err = enc.Encode(result)
		if err != nil {
			return fmt.Errorf("Error encoding result : " + err.Error())
		}
		return nil
	})

	endpoint.AddHandleFunc("REVOKE", func(rw *bufio.ReadWriter) error {
		var payload http.AuthorizationAndLicense
		dec := gob.NewDecoder(rw)
		err := dec.Decode(&payload)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("Missing mandatory payload.")
			}
			return fmt.Errorf("Error decoding result : " + err.Error())
		}
		if !server.Auth(payload.User, payload.Password) {
			return fmt.Errorf("Error : bad username / password (`" + payload.User + "`:`" + payload.Password + "`)")
		}
		//server.LogInfo("Received GOB License : %#v", payload.License)
		server.LogInfo("Revoking license with id %q", payload.License.Id)
		licenseStatus := &model.LicenseStatus{Status: model.StatusRevoked}
		_, err = LendingCancellation(server, licenseStatus, ParamKey{payload.License.Id})
		if err != nil {
			return err
		}
		result, err := server.Store().LicenseStatus().GetByLicenseId(payload.License.Id)
		if err != nil {
			return err
		}
		enc := gob.NewEncoder(rw)
		err = enc.Encode(result)
		if err != nil {
			return fmt.Errorf("Error encoding result : " + err.Error())
		}
		return nil
	})
	go func() {
		// Start listening.
		endpoint.Listen(":9000")
	}()
}
