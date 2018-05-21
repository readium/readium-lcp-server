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
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/controller/common"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
)

// CreateLicenseStatusDocument creates a license status and adds it to database
// It is triggered by a notification from the license server
//
func CreateLicenseStatusDocument(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	payload, err := common.ReadLicensePayload(req)

	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	var ls *model.LicenseStatus
	ls.MakeLicenseStatus(payload, server.Config().LicenseStatus.Register, server.Config().LicenseStatus.RentingDays)

	err = server.Store().LicenseStatus().Add(ls)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	resp.WriteHeader(http.StatusCreated)
}

// GetLicenseStatusDocument gets a license status from the db by license id
// checks potential_rights_end and fill it
//
func GetLicenseStatusDocument(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	vars := mux.Vars(req)

	licenseID := vars["key"]

	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			server.NotFoundHandler()(resp, req)
			logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusNotFound), "License id not found")
			return
		}

		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	currentDateTime := time.Now().UTC().Truncate(time.Second)

	// if a rights end date is set, check if the license has expired
	if licenseStatus.CurrentEndLicense.Valid {
		diff := currentDateTime.Sub(licenseStatus.CurrentEndLicense.Time)

		// if the rights end date has passed for a ready or active license
		if (diff > 0) && ((licenseStatus.Status == model.StatusActive) || (licenseStatus.Status == model.StatusReady)) {
			// the license has expired
			licenseStatus.Status = model.StatusExpired
			// update the db
			err = server.Store().LicenseStatus().Update(licenseStatus)
			if err != nil {
				server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
				logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
				return
			}
		}
	}

	err = fillLicenseStatus(licenseStatus, req, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	resp.Header().Set(common.HdrContentType, common.ContentTypeLsdJson)

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false
	enc := json.NewEncoder(resp)
	// write the JSON encoding of the license status to the stream, followed by a newline character
	err = enc.Encode(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// log the event in the compliance log
	// log the user agent of the caller
	msg := licenseStatus.Status.String() + " - agent: " + req.UserAgent()
	logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusOK), msg)
}

// RegisterDevice registers a device for a given license,
// using the device id &  name as  parameters;
// returns the updated license status
//
func RegisterDevice(resp http.ResponseWriter, req *http.Request, server common.IServer) {

	resp.Header().Set(common.HdrContentType, common.ContentTypeLsdJson)
	vars := mux.Vars(req)

	var msg string

	// get the license id from the url
	licenseID := vars["key"]
	// check the existence of the license in the lsd server
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			// the license is not stored in the lsd server
			msg = "The license id " + licenseID + " was not found in the database"
			server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusNotFound})
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return
		}
		// unknown error
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), "")
		return
	}

	deviceID := req.FormValue("id")
	deviceName := req.FormValue("name")

	dILen := len(deviceID)
	dNLen := len(deviceName)

	// check the mandatory request parameters
	if (dILen == 0) || (dILen > 255) || (dNLen == 0) || (dNLen > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// in case we want to test the resilience of an app to registering failures
	if server.GoofyMode() {
		msg = "**goofy mode** registering error"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// check the status of the license.
	// the device cannot be registered if the license has been revoked, returned, cancelled or expired
	if (licenseStatus.Status != model.StatusActive) && (licenseStatus.Status != model.StatusReady) {
		msg = "License is neither ready or active"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// check if the device has already been registered for this license
	deviceStatus, err := server.Store().Transaction().CheckDeviceStatus(licenseStatus.Id, deviceID)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	if deviceStatus != "" { // this is not considered a server side error, even if the spec states that devices must not do it.
		server.LogInfo("The device with id %v and name %v has already been registered"+deviceID, deviceName)
		// a status document will be sent back to the caller

	} else {

		// create a registered event
		event := makeEvent(model.StatusActive, deviceName, deviceID, licenseStatus.Id)
		err = server.Store().Transaction().Add(event)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}

		// the license has been updated, the corresponding field is set
		licenseStatus.StatusUpdated = model.NewTime(event.Timestamp, true)

		// license status set to active if it was ready
		if licenseStatus.Status == model.StatusReady {
			licenseStatus.Status = model.StatusActive
		}
		// one more device attached to this license
		licenseStatus.DeviceCount.Int64++

		// update the license status in db
		err = server.Store().LicenseStatus().Update(licenseStatus)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}
		// log the event in the compliance log
		msg = "device name: " + deviceName + "  id: " + deviceID + "  new count: " + strconv.Itoa(int(licenseStatus.DeviceCount.Int64))
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusOK), msg)

	} // the device has just registered this license

	// the device has registered the license (now *or before*)
	// fill the updated license status
	err = fillLicenseStatus(licenseStatus, req, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// the device count must not be sent back to the caller
	licenseStatus.DeviceCount.Valid = false
	// send back the license status to the caller
	enc := json.NewEncoder(resp)
	err = enc.Encode(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// LendingReturn checks that the calling device is activated, then modifies
// the end date associated with the given license & returns updated and filled license status
//
func LendingReturn(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	resp.Header().Set(common.HdrContentType, common.ContentTypeLsdJson)
	vars := mux.Vars(req)
	licenseID := vars["key"]

	var msg string

	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + licenseID + " was not found in the database"
			server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusNotFound})
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return
		}

		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), "")
		return
	}

	deviceID := req.FormValue("id")
	deviceName := req.FormValue("name")

	// check request parameters
	if (len(deviceName) > 255) || (len(deviceID) > 255) {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
		return
	}

	// check & set the status of the license status according to its current value
	switch licenseStatus.Status {
	case model.StatusReady:
		licenseStatus.Status = model.StatusCancelled
		break
	case model.StatusActive:
		licenseStatus.Status = model.StatusReturned
		break
	default:
		msg = "The current license status is " + licenseStatus.Status.String() + "; return forbidden"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// create a return event
	event := makeEvent(model.StatusReturned, deviceName, deviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := notifyLCPServer(event.Timestamp, licenseID, server)
	if errorr != nil {
		server.Error(resp, req, common.Problem{Detail: errorr.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		server.Error(resp, req, common.Problem{Detail: errorr.Error(), Status: httpStatusCode})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(httpStatusCode), err.Error())
		return
	}

	licenseStatus.CurrentEndLicense = model.NewTime(event.Timestamp, true)
	// update the license status
	licenseStatus.StatusUpdated = model.NewTime(event.Timestamp, true)
	licenseStatus.LicenseUpdated = model.NewTime(event.Timestamp, true)

	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	msg = "device name: " + deviceName + "  id: " + deviceID
	logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusOK), msg)

	// fill the license status
	err = fillLicenseStatus(licenseStatus, req, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false
	enc := json.NewEncoder(resp)
	err = enc.Encode(licenseStatus)

	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// LendingRenewal checks that the calling device is registered with the license,
// then modifies the end date associated with the license
// and returns an updated license status to the caller.
// the 'end' parameter is optional; if absent, the end date is computed from
// the current end date plus a configuration parameter.
// Note: as per the spec, a non-registered device can renew a loan.
//
func LendingRenewal(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	resp.Header().Set(common.HdrContentType, common.ContentTypeLsdJson)
	vars := mux.Vars(req)

	var msg string

	// get the license status by license id
	licenseID := vars["key"]
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)

	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + licenseID + " was not found in the database"
			server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusNotFound})
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return
		}
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	deviceID := req.FormValue("id")
	deviceName := req.FormValue("name")

	// check the request parameters
	if (len(deviceName) > 255) || (len(deviceID) > 255) {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
		return
	}
	// check that the license status is active.
	// note: renewing an unactive (ready) license is forbidden
	if licenseStatus.Status != model.StatusActive {
		msg = "The current license status is " + licenseStatus.Status.String() + "; renew forbidden"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// check if the license contains a date end property
	var currentEnd time.Time
	if !licenseStatus.CurrentEndLicense.Valid || (licenseStatus.CurrentEndLicense.Time).IsZero() {
		msg = "This license has no current end date; it cannot be renewed"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return
	}
	currentEnd = licenseStatus.CurrentEndLicense.Time
	server.LogInfo("Lending renewal. Current end date ", currentEnd.UTC().Format(time.RFC3339))

	var suggestedEnd time.Time
	// check if the 'end' request parameter is empty
	timeEndString := req.FormValue("end")
	if timeEndString == "" {
		// get the config  parameter renew_days
		renewDays := server.Config().LicenseStatus.RenewDays
		if renewDays == 0 {
			msg = "No explicit end value and no configured value"
			server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusInternalServerError})
			logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), msg)
			return
		}
		// compute a suggested duration from the config value
		var suggestedDuration time.Duration
		suggestedDuration = 24 * time.Hour * time.Duration(renewDays) // nanoseconds

		// compute the suggested end date from the current end date
		suggestedEnd = currentEnd.Add(time.Duration(suggestedDuration))
		server.LogInfo("Default extension request until ", suggestedEnd.UTC().Format(time.RFC3339))

		// if the 'end' request parameter is set
	} else {
		var err error
		suggestedEnd, err = time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
			return
		}
		server.LogInfo("Explicit extension request until ", suggestedEnd.UTC().Format(time.RFC3339))
	}

	// check the suggested end date vs the upper end date (which is already set in our implementation)
	if suggestedEnd.After(licenseStatus.PotentialRightsEnd.Time) {
		msg := "Attempt to renew with a date greater than potential rights end = " + licenseStatus.PotentialRightsEnd.Time.UTC().Format(time.RFC3339)
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return
	}
	// check the suggested end date vs the current end date
	if suggestedEnd.Before(currentEnd) {
		msg := "Attempt to renew with a date before the current end date"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusForbidden})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// create a renew event
	event := makeEvent(model.EventRenewed, deviceName, deviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := notifyLCPServer(suggestedEnd, licenseID, server)
	if errorr != nil {
		server.Error(resp, req, common.Problem{Detail: errorr.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), errorr.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		server.Error(resp, req, common.Problem{Detail: errorr.Error(), Status: httpStatusCode})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(httpStatusCode), errorr.Error())
		return
	}
	// update the license status fields
	licenseStatus.Status = model.StatusActive
	licenseStatus.CurrentEndLicense = model.NewTime(suggestedEnd, true)
	licenseStatus.StatusUpdated = model.NewTime(event.Timestamp, true)
	licenseStatus.LicenseUpdated = model.NewTime(event.Timestamp, true)

	// update the license status in db
	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	msg = "new end date: " + suggestedEnd.UTC().Format(time.RFC3339)
	logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusOK), msg)

	// fill the localized 'message', the 'links' and 'event' objects in the license status
	err = fillLicenseStatus(licenseStatus, req, server)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// return the updated license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false
	enc := json.NewEncoder(resp)
	err = enc.Encode(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// FilterLicenseStatuses returns a sequence of license statuses, in their id order
// function for detecting licenses which used a lot of devices
//
func FilterLicenseStatuses(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	resp.Header().Set(common.HdrContentType, common.ContentTypeJson)

	// Get request parameters. If not defined, set default values
	rDevices := req.FormValue("devices")
	if rDevices == "" {
		rDevices = "1"
	}

	rPage := req.FormValue("page")
	if rPage == "" {
		rPage = "1"
	}

	rPerPage := req.FormValue("per_page")
	if rPerPage == "" {
		rPerPage = "10"
	}

	devicesLimit, err := strconv.ParseInt(rDevices, 10, 32)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	page, err := strconv.ParseInt(rPage, 10, 32)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	perPage, err := strconv.ParseInt(rPerPage, 10, 32)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}

	if (page < 1) || (perPage < 1) || (devicesLimit < 1) {
		server.Error(resp, req, common.Problem{Detail: "Devices, page, per_page must be positive number", Status: http.StatusBadRequest})
		return
	}

	page--

	server.LogInfo("Pagination : %d %d %d", devicesLimit, perPage, page*perPage)

	licenseStatuses, err := server.Store().LicenseStatus().List(devicesLimit, perPage, page*perPage)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	devices := strconv.Itoa(int(devicesLimit))
	lsperpage := strconv.Itoa(int(perPage) + 1)
	var resultLink string

	if len(licenseStatuses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		resultLink += "</licenses/?devices=" + devices + "&page=" + nextPage + "&per_page=" + lsperpage + ">; rel=\"next\"; title=\"next\""
	}

	if page > 0 {
		previousPage := strconv.Itoa(int(page) - 1)
		if len(resultLink) > 0 {
			resultLink += ", "
		}
		resultLink += "</licenses/?devices=" + devices + "&page=" + previousPage + "&per_page=" + lsperpage + ">; rel=\"previous\"; title=\"previous\""
	}

	if len(resultLink) > 0 {
		resp.Header().Set("LicenseLink", resultLink)
	}

	enc := json.NewEncoder(resp)
	err = enc.Encode(licenseStatuses)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
}

// ListRegisteredDevices returns data about the use of a given license
//
func ListRegisteredDevices(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	resp.Header().Set(common.HdrContentType, common.ContentTypeJson)

	vars := mux.Vars(req)
	licenseID := vars["key"]

	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			server.NotFoundHandler()(resp, req)
			//logger.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusNotFound))
			return
		}

		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	registeredDevicesList, err := server.Store().Transaction().ListRegisteredDevices(licenseStatus.Id)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	enc := json.NewEncoder(resp)
	err = enc.Encode(registeredDevicesList)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
}

// LendingCancellation cancels (before use) or revokes (after use)  a license.
// parameters:
//	key: license id
//	partial license status: the new status and a message indicating why the status is being changed
//	The new status can be either STATUS_CANCELLED or STATUS_REVOKED
//
func LendingCancellation(resp http.ResponseWriter, req *http.Request, server common.IServer) {
	// get the license id
	vars := mux.Vars(req)
	licenseID := vars["key"]
	// get the current license status
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(licenseID)
	if err != nil {
		// erroneous license id
		if licenseStatus == nil {
			server.NotFoundHandler()(resp, req)
			logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusNotFound), "License id not found")
			return
		}
		// other error
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// get the partial license status document
	newStatus, err := common.ReadLicenseStatusPayload(req)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// the new status must be either cancelled or revoked
	if newStatus.Status != model.StatusRevoked && newStatus.Status != model.StatusCancelled {
		msg := "The new status must be either cancelled or revoked"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// cancelling is only possible when the status is ready
	if newStatus.Status == model.StatusCancelled && licenseStatus.Status != model.StatusReady {
		msg := "The license is not on ready state, it can't be cancelled"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// revocation is only possible when the status is ready or active
	if newStatus.Status == model.StatusRevoked && licenseStatus.Status != model.StatusReady && licenseStatus.Status != model.StatusActive {
		msg := "The license is not on ready or active state, it can't be revoked"
		server.Error(resp, req, common.Problem{Detail: msg, Status: http.StatusBadRequest})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// the new expiration time is now
	currentTime := model.TruncatedNow()

	// update the license with the new expiration time, via a call to the lcp Server
	httpStatusCode, erru := notifyLCPServer(currentTime.Time, licenseID, server)
	if erru != nil {
		server.Error(resp, req, common.Problem{Detail: erru.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), erru.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("License update notif to lcp server failed with http code " + strconv.Itoa(httpStatusCode))
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: httpStatusCode})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(httpStatusCode), err.Error())
		return
	}
	// create a cancel or revoke event
	curStatus := model.StatusRevoked

	if newStatus.Status == model.StatusCancelled {
		curStatus = model.StatusCancelled
	}
	// the event source is not a device.
	deviceName := "system"
	deviceID := "system"
	event := makeEvent(curStatus, deviceName, deviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// update the license status properties with the new status & expiration item (now)
	licenseStatus.Status = newStatus.Status
	licenseStatus.CurrentEndLicense = currentTime
	licenseStatus.StatusUpdated = currentTime
	licenseStatus.LicenseUpdated = currentTime

	// update the license status in db
	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		server.Error(resp, req, common.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// log
	logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusOK), "license "+curStatus.String()+"; Device count: "+strconv.Itoa(int(licenseStatus.DeviceCount.Int64)))
}
