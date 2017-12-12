// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilsd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/license_statuses"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/status"
	"github.com/readium/readium-lcp-server/transactions"
)

type Server interface {
	Transactions() transactions.Transactions
	LicenseStatuses() licensestatuses.LicenseStatuses
	GoofyMode() bool
}

// CreateLicenseStatusDocument creates a license status and adds it to database
// It is triggered by a notification from the license server
//
func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	var lic license.License
	err := apilcp.DecodeJSONLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	var ls licensestatuses.LicenseStatus
	makeLicenseStatus(lic, &ls)

	err = s.LicenseStatuses().Add(ls)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
}

// GetLicenseStatusDocument gets a license status from the db by license id
// checks potential_rights_end and fill it
//
func GetLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseID := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusNotFound), "License id not found")
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	currentDateTime := time.Now().UTC().Truncate(time.Second)

	// if the potential rights end is set
	if licenseStatus.PotentialRights != nil && licenseStatus.PotentialRights.End != nil && !(*licenseStatus.PotentialRights.End).IsZero() {
		diff := currentDateTime.Sub(*(licenseStatus.PotentialRights.End))

		// if the potential rights end has passed for a ready or active license
		if (diff > 0) && ((licenseStatus.Status == status.STATUS_ACTIVE) || (licenseStatus.Status == status.STATUS_READY)) {
			// the license has expired
			licenseStatus.Status = status.STATUS_EXPIRED
			// update the db
			err = s.LicenseStatuses().Update(*licenseStatus)
			if err != nil {
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
				logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusInternalServerError), err.Error())
				return
			}
		}
	}

	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	// write the JSON encoding of the license status to the stream, followed by a newline character
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// log the event in the compliance log
	// log the user agent of the caller
	msg := licenseStatus.Status + " - agent: " + r.UserAgent()
	logging.WriteToFile(complianceTestNumber, LICENSE_STATUS, strconv.Itoa(http.StatusOK), msg)
}

// RegisterDevice registers a device for a given license,
// using the device id &  name as  parameters;
// returns the updated license status
//
func RegisterDevice(w http.ResponseWriter, r *http.Request, s Server) {

	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	var msg string

	// get the license id from the url
	licenseID := vars["key"]
	// check the existence of the license in the lsd server
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			// the license is not stored in the lsd server
			msg = "The license id " + licenseID + " was not found in the database"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusNotFound)
			logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusNotFound), msg)
			return
		}
		// unknown error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), "")
		return
	}

	deviceId := r.FormValue("id")
	deviceName := r.FormValue("name")

	dILen := len(deviceId)
	dNLen := len(deviceName)

	// check the mandatory request parameters
	if (dILen == 0) || (dILen > 255) || (dNLen == 0) || (dNLen > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// in case we want to test the resilience of an app to registering failures
	if s.GoofyMode() {
		msg = "**goofy mode** registering error"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// check the status of the license.
	// the device cannot be registered if the license has been revoked, returned, cancelled or expired
	if (licenseStatus.Status != status.STATUS_ACTIVE) && (licenseStatus.Status != status.STATUS_READY) {
		msg = "License is neither ready or active"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// check if the device has already been registered for this license
	deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	if deviceStatus != "" { // this is  considered  an error, as per the lsd spec
		msg = "The device with id " + deviceId + " and name " + deviceName + " has already been registered"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusForbidden), msg)
		return

	} else {

		// create a registered event
		event := makeEvent(status.STATUS_ACTIVE, deviceName, deviceId, licenseStatus.Id)
		err = s.Transactions().Add(*event, status.STATUS_ACTIVE_INT)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}

		// the license has been updated, the corresponding field is set
		licenseStatus.Updated.Status = &event.Timestamp

		// license status set to active if it was ready
		if licenseStatus.Status == status.STATUS_READY {
			licenseStatus.Status = status.STATUS_ACTIVE
		}
		// one more device attached to this license
		*licenseStatus.DeviceCount++

		// update the license status in db
		err = s.LicenseStatuses().Update(*licenseStatus)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}
		// log the event in the compliance log
		msg = "device name: " + deviceName + "  id: " + deviceId + "  new count: " + strconv.Itoa(*licenseStatus.DeviceCount)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusOK), msg)

	} // the device has just registered this license

	// the device has registered the license (now *or before*)
	// fill the updated license status
	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// send the license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// LendingReturn checks that the calling device is activated, then modifies
// the end date associated with the given license & returns updated and filled license status
//
func LendingReturn(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)
	licenseID := vars["key"]

	var msg string

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + licenseID + " was not found in the database"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusNotFound)
			logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusNotFound), msg)
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), "")
		return
	}

	deviceId := r.FormValue("id")
	deviceName := r.FormValue("name")

	// check request parameters
	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusBadRequest), err.Error())
		return
	}

	// check & set the status of the license status according to its current value
	switch licenseStatus.Status {
	case status.STATUS_READY:
		licenseStatus.Status = status.STATUS_CANCELLED
		break
	case status.STATUS_ACTIVE:
		licenseStatus.Status = status.STATUS_RETURNED
		break
	default:
		msg = "The current license status is " + licenseStatus.Status + "; return forbidden"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// check if the device has already been registered for this license
	var deviceStatus string
	if deviceId != "" {
		var err error
		deviceStatus, err = s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}
	} else {
		deviceStatus = ""
	}
	// refuse to return a license if the device has not been registered.
	// this is a security against rogue returns
	if deviceStatus == "" {
		msg = "This device has never been registered for this license; return forbidden"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// create a return event
	event := makeEvent(status.STATUS_RETURNED, deviceName, deviceId, licenseStatus.Id)
	err = s.Transactions().Add(*event, status.STATUS_RETURNED_INT)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := updateLicense(event.Timestamp, licenseID)
	if errorr != nil {
		problem.Error(w, r, problem.Problem{Detail: errorr.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		problem.Error(w, r, problem.Problem{Detail: errorr.Error()}, httpStatusCode)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(httpStatusCode), err.Error())
		return
	}
	licenseStatus.CurrentEndLicense = &event.Timestamp

	// update the license status
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	msg = "device name: " + deviceName + "  id: " + deviceId
	logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusOK), msg)

	// fill the license status
	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// LendingRenewal checks that the calling device is registered with the license,
// then modifies the end date associated with the license
// and returns an updated license status to the caller.
// the 'end' parameter is optional; if absent, the end date is computed from
// the current end date plus a configuration parameter.
//
func LendingRenewal(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	var msg string

	// get the license status by license id
	licenseID := vars["key"]
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)

	if err != nil {
		if licenseStatus == nil {
			msg = "The license id " + licenseID + " was not found in the database"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusNotFound)
			logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusNotFound), msg)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	deviceId := r.FormValue("id")
	deviceName := r.FormValue("name")

	// check the request parameters
	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusBadRequest), err.Error())
		return
	}
	// check that the license status is active.
	// note: renewing an unactive (ready) license is forbidden
	if licenseStatus.Status != status.STATUS_ACTIVE {
		msg = "The current license status is " + licenseStatus.Status + "; renew forbidden"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	// check if the device has already been registered for this license
	var deviceStatus string
	if deviceId != "" {
		var err error
		deviceStatus, err = s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}
	} else {
		deviceStatus = ""
	}
	// refuse to renew a license if the device has never been registered before.
	// this is a security against rogue renews
	if deviceStatus == "" {
		msg = "This device has never been registered for this license; renew forbidden"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, RETURN_LICENSE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}

	// check if the license contains a potential date end property (the max renew date)
	if licenseStatus.PotentialRights == nil || licenseStatus.PotentialRights.End == nil || (*licenseStatus.PotentialRights.End).IsZero() {
		msg = "This license has no upper date for the loan; may not be a loan license"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), msg)
		return
	}

	// set the new end date
	var suggestedEnd time.Time
	// if the 'end' request parameter is empty
	timeEndString := r.FormValue("end")
	if timeEndString == "" {
		// get the config  parameter renew_days
		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			msg = "No explicit end value and no configured value"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), msg)
			return
		}
		// compute a suggested duration from the config value
		var suggestedDuration time.Duration
		suggestedDuration = 24 * time.Hour * time.Duration(renewDays) // nanoseconds

		log.Print("Renew possible for " + strconv.Itoa(renewDays) + " days")

		// compute the suggested end date from the current end date
		if licenseStatus.CurrentEndLicense != nil && !(*licenseStatus.CurrentEndLicense).IsZero() {
			suggestedEnd = (*licenseStatus.CurrentEndLicense).Add(time.Duration(suggestedDuration))

			log.Print("Suggested extension until " + suggestedEnd.UTC().Format(time.RFC3339))

		} else {
			msg = "CurrentEndLicense for LSD License Status is not set"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), msg)
			return
		}
		// if the 'end' request parameter is set
	} else {
		end, err := time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
			return
		}
		// only there because the go compiler does not accept a direct affectation
		suggestedEnd = end
	}
	// check the suggested end date vs now and the upper end date
	if suggestedEnd.After(*licenseStatus.PotentialRights.End) {
		msg := "Attempt to renew with a date greater than potential rights end = " + licenseStatus.PotentialRights.End.UTC().Format(time.RFC3339)
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusForbidden), msg)
		return
	}

	if suggestedEnd.Before(time.Now()) {
		msg := "Attempt to renew with a date before now"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusForbidden)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusForbidden), msg)
		return
	}
	// create a renew event
	event := makeEvent(status.EVENT_RENEWED, deviceName, deviceId, licenseStatus.Id)
	err = s.Transactions().Add(*event, status.EVENT_RENEWED_INT)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	// update a license via a call to the lcp Server
	httpStatusCode, errorr := updateLicense(suggestedEnd, licenseID)
	if errorr != nil {
		problem.Error(w, r, problem.Problem{Detail: errorr.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), errorr.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		problem.Error(w, r, problem.Problem{Detail: errorr.Error()}, httpStatusCode)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(httpStatusCode), errorr.Error())
		return
	}
	// update the license status fields
	licenseStatus.Status = status.STATUS_ACTIVE
	licenseStatus.CurrentEndLicense = &suggestedEnd
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp

	// update the license status in db
	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}

	msg = "new end date: " + suggestedEnd.UTC().Format(time.RFC3339)
	logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusOK), msg)

	// fill the localized 'message', the 'links' and 'event' objects in the license status
	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// return the updated license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, RENEW_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
}

// FilterLicenseStatuses returns a sequence of license statuses, in their id order
// function for detecting licenses which used a lot of devices
//
func FilterLicenseStatuses(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_JSON)

	// Get request parameters. If not defined, set default values
	rDevices := r.FormValue("devices")
	if rDevices == "" {
		rDevices = "1"
	}

	rPage := r.FormValue("page")
	if rPage == "" {
		rPage = "1"
	}

	rPerPage := r.FormValue("per_page")
	if rPerPage == "" {
		rPerPage = "10"
	}

	devicesLimit, err := strconv.ParseInt(rDevices, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	page, err := strconv.ParseInt(rPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	perPage, err := strconv.ParseInt(rPerPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if (page < 1) || (perPage < 1) || (devicesLimit < 1) {
		problem.Error(w, r, problem.Problem{Detail: "Devices, page, per_page must be positive number"}, http.StatusBadRequest)
		return
	}

	page -= 1

	licenseStatuses := make([]licensestatuses.LicenseStatus, 0)

	fn := s.LicenseStatuses().List(devicesLimit, perPage, page*perPage)
	for it, err := fn(); err == nil; it, err = fn() {
		licenseStatuses = append(licenseStatuses, it)
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
		w.Header().Set("Link", resultLink)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatuses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// ListRegisteredDevices returns data about the use of a given license
//
func ListRegisteredDevices(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_JSON)

	vars := mux.Vars(r)
	licenseID := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			//logging.WriteToFile(complianceTestNumber, REGISTER_DEVICE, strconv.Itoa(http.StatusNotFound))
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	registeredDevicesList := transactions.RegisteredDevicesList{Devices: make([]transactions.Device, 0), Id: licenseStatus.LicenseRef}

	fn := s.Transactions().ListRegisteredDevices(licenseStatus.Id)
	for it, err := fn(); err == nil; it, err = fn() {
		registeredDevicesList.Devices = append(registeredDevicesList.Devices, it)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(registeredDevicesList)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// LendingCancellation cancels (before use) or revokes (after use)  a license
// parameters:
//	key: license id
//	partial license status: the new status and a message indicating why the status is being changed
//	The new status can be either STATUS_CANCELLED or STATUS_REVOKED
//
func LendingCancellation(w http.ResponseWriter, r *http.Request, s Server) {
	// get the license id
	vars := mux.Vars(r)
	licenseID := vars["key"]
	// get the current license status
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseID)
	if err != nil {
		// erroneous license id
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusNotFound), "License id not found")
			return
		}
		// other error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// get the partial license status document
	var newStatus licensestatuses.LicenseStatus
	err = decodeJsonLicenseStatus(r, &newStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// the new status must be either cancelled or revoked
	if newStatus.Status != status.STATUS_REVOKED && newStatus.Status != status.STATUS_CANCELLED {
		msg := "The new status must be either cancelled or revoked"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// cancelling is only possible when the status is ready
	if newStatus.Status == status.STATUS_CANCELLED && licenseStatus.Status != status.STATUS_READY {
		msg := "The license is not on ready state, it can't be cancelled"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// revocation is only possible when the status is ready or active
	if newStatus.Status == status.STATUS_REVOKED && licenseStatus.Status != status.STATUS_READY && licenseStatus.Status != status.STATUS_ACTIVE {
		msg := "The license is not on ready or active state, it an't be revoked"
		problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusBadRequest)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusBadRequest), msg)
		return
	}
	// the new expiration time is now
	currentTime := time.Now().UTC().Truncate(time.Second)

	// update the license with the new expiration time, via a call to the lcp Server
	httpStatusCode, erru := updateLicense(currentTime, licenseID)
	if erru != nil {
		problem.Error(w, r, problem.Problem{Detail: erru.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusInternalServerError), erru.Error())
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("License update notif to lcp server failed with http code " + strconv.Itoa(httpStatusCode))
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, httpStatusCode)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(httpStatusCode), err.Error())
		return
	}
	// create a cancel or revoke event
	var st string
	var ty int
	if newStatus.Status == status.STATUS_CANCELLED {
		st = status.STATUS_CANCELLED
		ty = status.STATUS_CANCELLED_INT
	} else {
		st = status.STATUS_REVOKED
		ty = status.STATUS_REVOKED_INT
	}
	// the event source is not a device.
	deviceName := "system"
	deviceId := "system"
	event := makeEvent(st, deviceName, deviceId, licenseStatus.Id)
	err = s.Transactions().Add(*event, ty)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// update the license status properties with the new status & expiration item (now)
	licenseStatus.Status = newStatus.Status
	licenseStatus.CurrentEndLicense = &currentTime
	licenseStatus.Updated.Status = &currentTime
	licenseStatus.Updated.License = &currentTime

	// update the license status in db
	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusInternalServerError), err.Error())
		return
	}
	// log
	logging.WriteToFile(complianceTestNumber, CANCEL_REVOKE_LICENSE, strconv.Itoa(http.StatusOK), "Device count: "+strconv.Itoa(*licenseStatus.DeviceCount))
}

// makeLicenseStatus sets fields of license status according to the config file
// and creates needed inner objects of license status
//
func makeLicenseStatus(license license.License, ls *licensestatuses.LicenseStatus) {
	ls.LicenseRef = license.Id

	registerAvailable := config.Config.LicenseStatus.Register

	if license.Rights == nil || license.Rights.End == nil {
		// The publication was purchased (not a loan), so we do not set LSD.PotentialRights.End
		ls.CurrentEndLicense = nil
	} else {
		// license.Rights.End exists => this is a loan
		endFromLicense := license.Rights.End.Add(0)
		ls.CurrentEndLicense = &endFromLicense
		ls.PotentialRights = new(licensestatuses.PotentialRights)

		rentingDays := config.Config.LicenseStatus.RentingDays
		if rentingDays > 0 {
			endFromConfig := license.Issued.Add(time.Hour * 24 * time.Duration(rentingDays))

			if endFromLicense.After(endFromConfig) {
				ls.PotentialRights.End = &endFromLicense
			} else {
				ls.PotentialRights.End = &endFromConfig
			}
		} else {
			ls.PotentialRights.End = &endFromLicense
		}
	}

	if registerAvailable {
		ls.Status = status.STATUS_READY
	} else {
		ls.Status = status.STATUS_ACTIVE
	}

	ls.Updated = new(licensestatuses.Updated)
	ls.Updated.License = &license.Issued

	currentTime := time.Now().UTC().Truncate(time.Second)
	ls.Updated.Status = &currentTime

	count := 0
	ls.DeviceCount = &count
}

// getEvents gets the events from database for the license status
//
func getEvents(ls *licensestatuses.LicenseStatus, s Server) error {
	events := make([]transactions.Event, 0)

	fn := s.Transactions().GetByLicenseStatusId(ls.Id)
	var err error
	var event transactions.Event
	for event, err = fn(); err == nil; event, err = fn() {
		events = append(events, event)
	}

	if err == transactions.NotFound {
		ls.Events = events
		err = nil
	}

	return err
}

// makeLinks creates and adds links to the license status
//
func makeLinks(ls *licensestatuses.LicenseStatus) {
	lsdBaseUrl := config.Config.LsdServer.PublicBaseUrl
	licenseLinkUrl := config.Config.LsdServer.LicenseLinkUrl
	lcpBaseUrl := config.Config.LcpServer.PublicBaseUrl
	//frontendBaseUrl := config.Config.FrontendServer.PublicBaseUrl
	registerAvailable := config.Config.LicenseStatus.Register

	licenseHasRightsEnd := ls.CurrentEndLicense != nil && !(*ls.CurrentEndLicense).IsZero()
	returnAvailable := config.Config.LicenseStatus.Return && licenseHasRightsEnd
	renewAvailable := config.Config.LicenseStatus.Renew && licenseHasRightsEnd

	links := new([]licensestatuses.Link)

	if licenseLinkUrl != "" {
		licenseLinkUrl_ := strings.Replace(licenseLinkUrl, "{license_id}", ls.LicenseRef, -1)
		link := licensestatuses.Link{Href: licenseLinkUrl_, Rel: "license", Type: api.ContentType_LCP_JSON, Templated: false}
		*links = append(*links, link)
	} else {
		link := licensestatuses.Link{Href: lcpBaseUrl + "/licenses/" + ls.LicenseRef, Rel: "license", Type: api.ContentType_LCP_JSON, Templated: false}
		*links = append(*links, link)
	}

	if registerAvailable {
		link := licensestatuses.Link{Href: lsdBaseUrl + "/licenses/" + ls.LicenseRef + "/register{?id,name}", Rel: "register", Type: api.ContentType_LSD_JSON, Templated: true}
		*links = append(*links, link)
	}

	if returnAvailable {
		link := licensestatuses.Link{Href: lsdBaseUrl + "/licenses/" + ls.LicenseRef + "/return{?id,name}", Rel: "return", Type: api.ContentType_LSD_JSON, Templated: true}
		*links = append(*links, link)
	}

	if renewAvailable {
		link := licensestatuses.Link{Href: lsdBaseUrl + "/licenses/" + ls.LicenseRef + "/renew{?end,id,name}", Rel: "renew", Type: api.ContentType_LSD_JSON, Templated: true}
		*links = append(*links, link)
	}

	ls.Links = *links
}

// makeEvent creates an event and fill it
//
func makeEvent(status string, deviceName string, deviceId string, licenseStatusFk int) *transactions.Event {
	event := transactions.Event{}
	event.DeviceId = deviceId
	event.DeviceName = deviceName
	event.Timestamp = time.Now().UTC().Truncate(time.Second)
	event.Type = status
	event.LicenseStatusFk = licenseStatusFk

	return &event
}

// decodeJsonLicenseStatus decodes license status json to the object
//
func decodeJsonLicenseStatus(r *http.Request, ls *licensestatuses.LicenseStatus) error {
	var dec *json.Decoder

	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_FORM_URL_ENCODED {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	err := dec.Decode(&ls)

	return err
}

// updateLicense updates a license by calling the License Server
// called from return, renew and cancel/revoke actions
//
func updateLicense(timeEnd time.Time, licenseID string) (int, error) {
	// get the lcp server url
	lcpBaseUrl := config.Config.LcpServer.PublicBaseUrl
	if len(lcpBaseUrl) <= 0 {
		return 0, errors.New("Undefined Config.LcpServer.PublicBaseUrl")
	}
	// create a minimum license object, limited to the license id plus rights
	// FIXME: remove the id (here and in the lcpserver license.go)
	minLicense := license.License{Id: licenseID, Rights: new(license.UserRights)}
	// set the new end date
	minLicense.Rights.End = &timeEnd

	var lcpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	// FIXME: this Pipe thing should be replaced by a json.Marshal
	pr, pw := io.Pipe()
	go func() {
		_ = json.NewEncoder(pw).Encode(minLicense)
		pw.Close()
	}()
	// prepare the request
	lcpURL := lcpBaseUrl + "/licenses/" + licenseID
	// message to the console
	log.Println("PATCH " + lcpURL)
	// send the content to the LCP server
	req, err := http.NewRequest("PATCH", lcpURL, pr)
	if err != nil {
		return 0, err
	}
	// set the credentials
	updateAuth := config.Config.LcpUpdateAuth
	if updateAuth.Username != "" {
		req.SetBasicAuth(updateAuth.Username, updateAuth.Password)
	}
	// set the content type
	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
	// send the request to the lcp server
	response, err := lcpClient.Do(req)
	if err == nil {
		if response.StatusCode != http.StatusOK {
			log.Println("Notify Lcp Server of License (" + licenseID + ") = " + strconv.Itoa(response.StatusCode))
		}
		return response.StatusCode, nil
	}

	log.Println("Error Notify Lcp Server of License (" + licenseID + "):" + err.Error())
	return 0, err
}

// fillLicenseStatus fills the localized 'message' field, the 'links' and 'event' objects in the license status
//
func fillLicenseStatus(ls *licensestatuses.LicenseStatus, r *http.Request, s Server) error {
	// add the localized message
	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &ls.Message, ls.Status)
	// add the links
	makeLinks(ls)
	// add the events
	err := getEvents(ls, s)

	return err
}
