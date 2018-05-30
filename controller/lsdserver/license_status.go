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
	"errors"
	"strconv"
	"time"

	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
)

// CreateLicenseStatusDocument creates a license status and adds it to database
// It is triggered by a notification from the license server
//
func CreateLicenseStatusDocument(server http.IServer, payload *model.License) (*string, error) {
	var ls *model.LicenseStatus
	ls.MakeLicenseStatus(payload, server.Config().LicenseStatus.Register, server.Config().LicenseStatus.RentingDays)

	err := server.Store().LicenseStatus().Add(ls)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	return nil, http.Problem{Status: http.StatusCreated, HttpHeaders: make(map[string][]string)}
}

// GetLicenseStatusDocument gets a license status from the db by license id
// checks potential_rights_end and fill it
//
func GetLicenseStatusDocument(server http.IServer, param ParamKey, hdr Headers) (*model.LicenseStatus, error) {
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)
	if err != nil {
		if licenseStatus == nil {
			logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusNotFound), "License id not found")
			return nil, http.Problem{Detail: "License " + param.Key + " not found", Status: http.StatusNotFound}
		}

		logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
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
				logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
	}

	err = fillLicenseStatus(licenseStatus, hdr, server)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false
	// log the event in the compliance log
	// log the user agent of the caller
	msg := licenseStatus.Status.String() + " - agent: " + hdr.UserAgent
	logger.WriteToFile(complianceTestNumber, LicenseStatus, strconv.Itoa(http.StatusOK), msg)
	return licenseStatus, nil
}

// RegisterDevice registers a device for a given license,
// using the device id &  name as  parameters;
// returns the updated license status
//
func RegisterDevice(server http.IServer, param ParamKeyAndDevice, hdr Headers) (*model.LicenseStatus, error) {
	var msg string

	// check the existence of the license in the lsd server
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)
	if err != nil {
		if licenseStatus == nil {
			// the license is not stored in the lsd server
			msg = "The license id " + param.Key + " was not found in the database"
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return nil, http.Problem{Detail: msg, Status: http.StatusNotFound}
		}
		// unknown error

		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), "")
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// check the mandatory request parameters
	if (len(param.DeviceID) == 0) || (len(param.DeviceID) > 255) || (len(param.DeviceName) == 0) || (len(param.DeviceName) > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"

		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusBadRequest), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusBadRequest}
	}

	// in case we want to test the resilience of an app to registering failures
	if server.GoofyMode() {
		msg = "**goofy mode** registering error"
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusBadRequest), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusBadRequest}
	}

	// check the status of the license.
	// the device cannot be registered if the license has been revoked, returned, cancelled or expired
	if (licenseStatus.Status != model.StatusActive) && (licenseStatus.Status != model.StatusReady) {
		msg = "License is neither ready or active"
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}

	// check if the device has already been registered for this license
	deviceStatus, err := server.Store().Transaction().CheckDeviceStatus(licenseStatus.Id, param.DeviceID)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	if deviceStatus != "" { // this is not considered a server side error, even if the spec states that devices must not do it.
		server.LogInfo("The device with id %v and name %v has already been registered"+param.DeviceID, param.DeviceName)
		// a status document will be sent back to the caller

	} else {
		// create a registered event
		event := makeEvent(model.StatusActive, param.DeviceName, param.DeviceID, licenseStatus.Id)
		err = server.Store().Transaction().Add(event)
		if err != nil {
			logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}

		// the license has been updated, the corresponding field is set
		licenseStatus.StatusUpdated = model.NewTime(event.Timestamp)

		// license status set to active if it was ready
		if licenseStatus.Status == model.StatusReady {
			licenseStatus.Status = model.StatusActive
		}
		// one more device attached to this license
		licenseStatus.DeviceCount.Int64++

		// update the license status in db
		err = server.Store().LicenseStatus().Update(licenseStatus)
		if err != nil {
			logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		// log the event in the compliance log
		msg = "device name: " + param.DeviceName + "  id: " + param.DeviceID + "  new count: " + strconv.Itoa(int(licenseStatus.DeviceCount.Int64))
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusOK), msg)

	} // the device has just registered this license

	// the device has registered the license (now *or before*)
	// fill the updated license status
	err = fillLicenseStatus(licenseStatus, hdr, server)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, RegistDevice, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// the device count must not be sent back to the caller
	licenseStatus.DeviceCount.Valid = false
	return licenseStatus, nil
}

// LendingReturn checks that the calling device is activated, then modifies
// the end date associated with the given license & returns updated and filled license status
//
func LendingReturn(server http.IServer, param ParamKeyAndDevice, hdr Headers) (*model.LicenseStatus, error) {
	var msg string

	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)
	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + param.Key + " was not found in the database"
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return nil, http.Problem{Detail: msg, Status: http.StatusNotFound}
		}

		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), "")
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	deviceID := param.DeviceID
	deviceName := param.DeviceName

	// check request parameters
	if (len(deviceName) > 255) || (len(deviceID) > 255) {
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
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
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}

	// create a return event
	event := makeEvent(model.StatusReturned, deviceName, deviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := notifyLCPServer(event.Timestamp, param.Key, server)
	if errorr != nil {
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: errorr.Error(), Status: http.StatusInternalServerError}
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(httpStatusCode), err.Error())
		return nil, http.Problem{Detail: errorr.Error(), Status: httpStatusCode}
	}

	licenseStatus.CurrentEndLicense = model.NewTime(event.Timestamp)
	// update the license status
	licenseStatus.StatusUpdated = model.NewTime(event.Timestamp)
	licenseStatus.LicenseUpdated = model.NewTime(event.Timestamp)

	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	msg = "device name: " + deviceName + "  id: " + deviceID
	logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusOK), msg)

	// fill the license status
	err = fillLicenseStatus(licenseStatus, hdr, server)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false
	return licenseStatus, nil
}

// LendingRenewal checks that the calling device is registered with the license,
// then modifies the end date associated with the license
// and returns an updated license status to the caller.
// the 'end' parameter is optional; if absent, the end date is computed from
// the current end date plus a configuration parameter.
// Note: as per the spec, a non-registered device can renew a loan.
//
func LendingRenewal(server http.IServer, param ParamKeyAndDevice, hdr Headers) (*model.LicenseStatus, error) {
	var msg string

	// get the license status by license id
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)

	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + param.Key + " was not found in the database"
			logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusNotFound), msg)
			return nil, http.Problem{Detail: msg, Status: http.StatusNotFound}
		}
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// check the request parameters
	if (len(param.DeviceName) > 255) || (len(param.DeviceID) > 255) {
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// check that the license status is active.
	// note: renewing an unactive (ready) license is forbidden
	if licenseStatus.Status != model.StatusActive {
		msg = "The current license status is " + licenseStatus.Status.String() + "; renew forbidden"
		logger.WriteToFile(complianceTestNumber, ReturnLicense, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}

	// check if the license contains a date end property
	var currentEnd time.Time
	if !licenseStatus.CurrentEndLicense.Valid || (licenseStatus.CurrentEndLicense.Time).IsZero() {
		msg = "This license has no current end date; it cannot be renewed"
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}
	currentEnd = licenseStatus.CurrentEndLicense.Time
	server.LogInfo("Lending renewal. Current end date ", currentEnd.UTC().Format(time.RFC3339))

	var suggestedEnd time.Time
	// check if the 'end' request parameter is empty
	timeEndString := param.End
	if timeEndString == "" {
		// get the config  parameter renew_days
		renewDays := server.Config().LicenseStatus.RenewDays
		if renewDays == 0 {
			msg = "No explicit end value and no configured value"
			logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), msg)
			return nil, http.Problem{Detail: msg, Status: http.StatusInternalServerError}
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
			logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusBadRequest), err.Error())
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
		}
		server.LogInfo("Explicit extension request until ", suggestedEnd.UTC().Format(time.RFC3339))
	}

	// check the suggested end date vs the upper end date (which is already set in our implementation)
	if suggestedEnd.After(licenseStatus.PotentialRightsEnd.Time) {
		msg := "Attempt to renew with a date greater than potential rights end = " + licenseStatus.PotentialRightsEnd.Time.UTC().Format(time.RFC3339)
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}
	// check the suggested end date vs the current end date
	if suggestedEnd.Before(currentEnd) {
		msg := "Attempt to renew with a date before the current end date"
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusForbidden), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusForbidden}
	}

	// create a renew event
	event := makeEvent(model.EventRenewed, param.DeviceName, param.DeviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := notifyLCPServer(suggestedEnd, param.Key, server)
	if errorr != nil {
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), errorr.Error())
		return nil, http.Problem{Detail: errorr.Error(), Status: http.StatusInternalServerError}
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(httpStatusCode), errorr.Error())
		return nil, http.Problem{Detail: errorr.Error(), Status: httpStatusCode}
	}
	// update the license status fields
	licenseStatus.Status = model.StatusActive
	licenseStatus.CurrentEndLicense = model.NewTime(suggestedEnd)
	licenseStatus.StatusUpdated = model.NewTime(event.Timestamp)
	licenseStatus.LicenseUpdated = model.NewTime(event.Timestamp)

	// update the license status in db
	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	msg = "new end date: " + suggestedEnd.UTC().Format(time.RFC3339)
	logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusOK), msg)

	// fill the localized 'message', the 'links' and 'event' objects in the license status
	err = fillLicenseStatus(licenseStatus, hdr, server)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, RenewLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// return the updated license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount.Valid = false

	return licenseStatus, nil
}

// FilterLicenseStatuses returns a sequence of license statuses, in their id order
// function for detecting licenses which used a lot of devices
//
func FilterLicenseStatuses(server http.IServer, param ParamDevicesAndPage) (model.LicensesStatusCollection, error) {
	if param.Devices == "" {
		param.Devices = "1"
	}
	// Get request parameters
	devicesLimit, err := strconv.ParseInt(param.Devices, 10, 32)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	// Count
	noOfLicenses, err := server.Store().LicenseStatus().Count(devicesLimit)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	if noOfLicenses == 0 {
		return nil, http.Problem{Detail: "No licenses statuses found for devices limit " + param.Devices, Status: http.StatusNotFound}
	}
	// Pagination
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfLicenses)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}
	// List them
	result, err := server.Store().LicenseStatus().List(devicesLimit, perPage, page*perPage)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// Result
	nonErr := http.Problem{Status: http.StatusOK, HttpHeaders: make(map[string][]string)}
	nonErr.HttpHeaders.Set("Link", http.MakePaginationHeader("http://localhost:"+strconv.Itoa(server.Config().LcpServer.Port)+"/licenses/?devices="+strconv.Itoa(int(devicesLimit)), page+1, perPage, noOfLicenses))
	return result, nonErr
}

// ListRegisteredDevices returns data about the use of a given license
//
func ListRegisteredDevices(server http.IServer, param ParamKey) (model.TransactionEventsCollection, error) {
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)
	if err != nil {
		if licenseStatus == nil {
			return nil, http.Problem{Detail: "not found.", Status: http.StatusNotFound}
		}
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}

	registeredDevicesList, err := server.Store().Transaction().ListRegisteredDevices(licenseStatus.Id)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	return registeredDevicesList, nil
}

// LendingCancellation cancels (before use) or revokes (after use)  a license.
// parameters:
//	key: license id
//	partial license status: the new status and a message indicating why the status is being changed
//	The new status can be either STATUS_CANCELLED or STATUS_REVOKED
//
func LendingCancellation(server http.IServer, payload *model.LicenseStatus, param ParamKey) (*string, error) {
	// get the current license status
	licenseStatus, err := server.Store().LicenseStatus().GetByLicenseId(param.Key)
	if err != nil {
		// erroneous license id
		if licenseStatus == nil {
			logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusNotFound), "License id not found")
			return nil, http.Problem{Detail: "license " + param.Key + " not found.", Status: http.StatusNotFound}
		}
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		// other error
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// the new status must be either cancelled or revoked
	if payload.Status != model.StatusRevoked && payload.Status != model.StatusCancelled {
		msg := "The new status must be either cancelled or revoked"
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusBadRequest}
	}
	// cancelling is only possible when the status is ready
	if payload.Status == model.StatusCancelled && licenseStatus.Status != model.StatusReady {
		msg := "The license is not on ready state, it can't be cancelled"
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusBadRequest}
	}
	// revocation is only possible when the status is ready or active
	if payload.Status == model.StatusRevoked && licenseStatus.Status != model.StatusReady && licenseStatus.Status != model.StatusActive {
		msg := "The license is not on ready or active state, it can't be revoked"
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusBadRequest), msg)
		return nil, http.Problem{Detail: msg, Status: http.StatusBadRequest}
	}
	// the new expiration time is now
	currentTime := model.TruncatedNow()

	// update the license with the new expiration time, via a call to the lcp Server
	httpStatusCode, erru := notifyLCPServer(currentTime.Time, param.Key, server)
	if erru != nil {
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), erru.Error())
		return nil, http.Problem{Detail: erru.Error(), Status: http.StatusInternalServerError}
	}

	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("License update notif to lcp server failed with http code " + strconv.Itoa(httpStatusCode))
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(httpStatusCode), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: httpStatusCode}
	}
	// create a cancel or revoke event
	curStatus := model.StatusRevoked

	if payload.Status == model.StatusCancelled {
		curStatus = model.StatusCancelled
	}
	// the event source is not a device.
	deviceName := "system"
	deviceID := "system"
	event := makeEvent(curStatus, deviceName, deviceID, licenseStatus.Id)
	err = server.Store().Transaction().Add(event)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// update the license status properties with the new status & expiration item (now)
	licenseStatus.Status = payload.Status
	licenseStatus.CurrentEndLicense = currentTime
	licenseStatus.StatusUpdated = currentTime
	licenseStatus.LicenseUpdated = currentTime

	// update the license status in db
	err = server.Store().LicenseStatus().Update(licenseStatus)
	if err != nil {
		logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	// log
	logger.WriteToFile(complianceTestNumber, CancelRevokeLicense, strconv.Itoa(http.StatusOK), "license "+curStatus.String()+"; Device count: "+strconv.Itoa(int(licenseStatus.DeviceCount.Int64)))
	return nil, nil
}
