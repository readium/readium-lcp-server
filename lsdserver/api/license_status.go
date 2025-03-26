// Copyright 2020 Readium Foundation. All rights reserved.
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
	"github.com/jtacoma/uritemplates"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	licensestatuses "github.com/readium/readium-lcp-server/license_statuses"
	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/status"
	"github.com/readium/readium-lcp-server/transactions"
)

// Server interface
type Server interface {
	Transactions() transactions.Transactions
	LicenseStatuses() licensestatuses.LicenseStatuses
	GoofyMode() bool
}

// CreateLicenseStatusDocument creates a license status and adds it to database
// It is triggered by a notification from the license server
func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	var lic license.License
	err := apilcp.DecodeJSONLicense(r, &lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	// add a log
	logging.Print("Create a Status Doc for License " + lic.ID)

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
func GetLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseID := vars["key"]

	// add a log
	logging.Print("Get a Status Doc for License " + licenseID)

	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	currentDateTime := time.Now().UTC().Truncate(time.Second)

	// if a rights end date is set, check if the license has expired
	if licenseStatus.CurrentEndLicense != nil {
		diff := currentDateTime.Sub(*(licenseStatus.CurrentEndLicense))

		// if the rights end date has passed for a ready or active license
		if (diff > 0) && ((licenseStatus.Status == status.STATUS_ACTIVE) || (licenseStatus.Status == status.STATUS_READY)) {
			// the license has expired
			licenseStatus.Status = status.STATUS_EXPIRED
			// set the updated status time
			currentTime := time.Now().UTC().Truncate(time.Second)
			licenseStatus.Updated.Status = &currentTime
			// update the db
			err = s.LicenseStatuses().Update(*licenseStatus)
			if err != nil {
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
				return
			}
		}
	}

	err = fillLicenseStatus(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
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
		return
	}
}

// RegisterDevice registers a device for a given license,
// using the device id &  name as  parameters;
// returns the updated license status
func RegisterDevice(w http.ResponseWriter, r *http.Request, s Server) {

	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	var msg string

	// get the license id from the url
	licenseID := vars["key"]

	deviceID := r.FormValue("id")
	deviceName := r.FormValue("name")

	// add a log
	logging.Print("Register the Device " + deviceName + " with id " + deviceID + " for License " + licenseID)

	dILen := len(deviceID)
	dNLen := len(deviceName)

	// check the mandatory request parameters
	if (dILen == 0) || (dILen > 255) || (dNLen == 0) || (dNLen > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"
		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// check the existence of the license in the lsd server
	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}
		// unknown error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// in case we want to test the resilience of an app to registering failures
	if s.GoofyMode() {
		msg = "**goofy mode** registering error"
		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// check the status of the license.
	// the device cannot be registered if the license has been revoked, returned, cancelled or expired
	if (licenseStatus.Status != status.STATUS_ACTIVE) && (licenseStatus.Status != status.STATUS_READY) {
		msg = "License is neither ready or active"
		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
		return
	}

	// check if the device has already been registered for this license
	deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.ID, deviceID)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	if deviceStatus != "" { // this is not considered a server side error, even if the spec states that devices must not do it.
		logging.Print("The Device has already been registered")
		// a status document will be sent back to the caller

	} else {

		// create a registered event
		event := makeEvent(status.STATUS_ACTIVE, deviceName, deviceID, licenseStatus.ID)
		err = s.Transactions().Add(*event, status.STATUS_ACTIVE_INT)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
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
			return
		}
		// add a log
		logging.Print("The new Device Count is " + strconv.Itoa(*licenseStatus.DeviceCount))

	} // the device has just been registered for this license

	// the device has been registered for the license (now *or before*)
	// fill the updated license status
	err = fillLicenseStatus(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// the device count must not be sent back to the caller
	licenseStatus.DeviceCount = nil
	// send back the license status to the caller
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// LendingReturn checks that the calling device is activated, then modifies
// the end date associated with the given license & returns updated and filled license status
func LendingReturn(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)
	licenseID := vars["key"]

	var msg string

	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	deviceID := r.FormValue("id")
	deviceName := r.FormValue("name")

	// check request parameters
	if (len(deviceName) > 255) || (len(deviceID) > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// add a log
	logging.Print("Return the Publication from Device " + deviceName + " with id " + deviceID + " for License " + licenseID)

	// check & set the status of the license status according to its current value
	switch licenseStatus.Status {
	case status.STATUS_READY:
		licenseStatus.Status = status.STATUS_CANCELLED
	case status.STATUS_ACTIVE:
		licenseStatus.Status = status.STATUS_RETURNED
	// a license can be returned even if it has expired, this is a final status.
	case status.STATUS_EXPIRED:
		licenseStatus.Status = status.STATUS_RETURNED
	case status.STATUS_RETURNED:
		msg = "The license has already been returned before"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_ALREADY, Detail: msg}, http.StatusForbidden)
		return
	default:
		msg = "The current license status is " + licenseStatus.Status + "; return forbidden"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
		return
	}

	// create a return event
	event := makeEvent(status.STATUS_RETURNED, deviceName, deviceID, licenseStatus.ID)
	err = s.Transactions().Add(*event, status.STATUS_RETURNED_INT)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// update a license via a call to the lcp Server
	// the event date is sent to the lcp server, covers the case where the lsd server clock is badly sync'd with the lcp server clock
	httpStatusCode, errorr := updateLicense(event.Timestamp, licenseID)
	if errorr != nil {
		problem.Error(w, r, problem.Problem{Detail: errorr.Error()}, http.StatusInternalServerError)
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		errorr = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: errorr.Error()}, httpStatusCode)
		return
	}
	licenseStatus.CurrentEndLicense = &event.Timestamp

	// update the license status
	licenseStatus.Updated.Status = &event.Timestamp
	// update the license updated timestamp with the event date
	licenseStatus.Updated.License = &event.Timestamp
	// remove the potential end timestamp
	licenseStatus.PotentialRights = nil

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// fill the license status
	err = fillLicenseStatus(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// LendingRenewal checks that the calling device is registered with the license,
// then modifies the end date associated with the license
// and returns an updated license status to the caller.
// Note: as per the spec, a non-registered device can renew a loan.
//
// parameters:
//
//	key: license id
//	end: the new end date for the license (optional)
func LendingRenewal(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	var msg string

	// get the license status by license id
	licenseID := vars["key"]

	deviceID := r.FormValue("id")
	deviceName := r.FormValue("name")

	// add a log
	logging.Print("Renew the Loan from Device " + deviceName + " with id " + deviceID + " for License " + licenseID)

	// check the request parameters
	if (len(deviceName) > 255) || (len(deviceID) > 255) {
		msg = "device id and device name are mandatory and their maximum length is 255 bytes"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// get the license status
	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// check the status of the license.
	// note: renewing an unactive (ready) license is forbidden
	if licenseStatus.Status == status.STATUS_EXPIRED {
		if !config.Config.LicenseStatus.RenewExpired {
			msg = "The license has expired and cannot be renewed"
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
			return
		}
	} else if licenseStatus.Status != status.STATUS_ACTIVE {
		msg = "The current license status is " + licenseStatus.Status + "; renew forbidden"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
		return
	}

	// check if the license contains a date end property
	if licenseStatus.CurrentEndLicense == nil || (*licenseStatus.CurrentEndLicense).IsZero() {
		msg = "This license has no current end date; it cannot be renewed"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
		return
	}
	currentEnd := *licenseStatus.CurrentEndLicense

	var suggestedEnd time.Time
	// check if the 'end' request parameter is empty
	timeEndString := r.FormValue("end")
	if timeEndString == "" {
		// get the config  parameter renew_days
		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			msg = "No explicit end value and no configured value"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusInternalServerError)
			return
		}
		// compute a suggested duration from the config value
		suggestedDuration := 24 * time.Hour * time.Duration(renewDays) // nanoseconds

		// compute the suggested end date from now
		if config.Config.LicenseStatus.RenewFromNow {
			suggestedEnd = time.Now().Add(time.Duration(suggestedDuration))
			// compute the suggested end date from the current end date
		} else {
			suggestedEnd = currentEnd.Add(time.Duration(suggestedDuration))
		}
		log.Print("Default extension request until ", suggestedEnd.UTC().Format(time.RFC3339))

		// if the 'end' request parameter is set
	} else {
		var err error
		suggestedEnd, err = time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		log.Print("Explicit extension request until ", suggestedEnd.UTC().Format(time.RFC3339))
	}

	// check the suggested end date vs the upper end date (which is already set in our implementation)
	//log.Print("Upper limit = ", licenseStatus.PotentialRights.End.UTC().Format(time.RFC3339))
	if suggestedEnd.After(*licenseStatus.PotentialRights.End) {
		msg := "Attempt to renew with a date greater than the upper limit = " + licenseStatus.PotentialRights.End.UTC().Format(time.RFC3339)
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: msg}, http.StatusForbidden)
		return
	}
	// check the suggested end date vs the current end date
	if suggestedEnd.Before(currentEnd) {
		msg := "Attempt to renew with a date before the current end date"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: msg}, http.StatusForbidden)
		return
	}

	// add a log
	logging.Print("Loan renewed until " + suggestedEnd.UTC().Format(time.RFC3339))

	// create a renew event
	event := makeEvent(status.EVENT_RENEWED, deviceName, deviceID, licenseStatus.ID)
	err = s.Transactions().Add(*event, status.EVENT_RENEWED_INT)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// update a license via a call to the lcp Server
	var httpStatusCode int
	httpStatusCode, err = updateLicense(suggestedEnd, licenseID)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: err.Error()}, httpStatusCode)
		return
	}
	// update the license status fields
	licenseStatus.Status = status.STATUS_ACTIVE
	licenseStatus.CurrentEndLicense = &suggestedEnd
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	//log.Print("Update timestamp ", event.Timestamp.UTC().Format(time.RFC3339))

	// update the license status in db
	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// fill the localized 'message', the 'links' and 'event' objects in the license status
	err = fillLicenseStatus(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// return the updated license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// ExtendSubscription extends the lifetime of a subscription license.
// It can re-activate an expired license (but not a returned or cancelled/revoked one);
// this allows the extension to be made after a trial period as ended.
//
// parameters:
//
//	key: license id
//	end: the new end date for the license (optional)
func ExtendSubscription(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	var msg string

	// get the license status by license id
	licenseID := vars["key"]

	// add a log
	logging.Print("Extend the Subscription for License " + licenseID)

	// get the license status
	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// the max end date must be set
	if licenseStatus.PotentialRights == nil || licenseStatus.PotentialRights.End == nil {
		msg := "The maximum end date must be set"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// extension is impossible if the status is revoked, cancelled or returned
	if licenseStatus.Status == status.STATUS_REVOKED || licenseStatus.Status == status.STATUS_CANCELLED || licenseStatus.Status == status.STATUS_RETURNED {
		msg := "The license cannot be extended as it is " + licenseStatus.Status
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// check if the license contains a date end property
	var currentEnd time.Time
	if licenseStatus.CurrentEndLicense == nil || (*licenseStatus.CurrentEndLicense).IsZero() {
		msg = "This license has no current end date; it cannot be extended"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: msg}, http.StatusForbidden)
		return
	}
	currentEnd = *licenseStatus.CurrentEndLicense
	log.Print("Current end date " + currentEnd.UTC().Format(time.RFC3339))
	if licenseStatus.Status == status.STATUS_EXPIRED {
		log.Println("This license had expired and will be re-activated")
	}

	var suggestedEnd time.Time
	// check if the 'end' request parameter is empty
	timeEndString := r.FormValue("end")
	if timeEndString == "" {
		// get the config parameter renew_days
		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			msg = "No explicit end value and no configured value"
			problem.Error(w, r, problem.Problem{Detail: msg}, http.StatusInternalServerError)
			return
		}
		// compute a suggested duration from the config value
		suggestedDuration := 24 * time.Hour * time.Duration(renewDays) // nanoseconds

		// compute the suggested end date from the current end date
		suggestedEnd = currentEnd.Add(time.Duration(suggestedDuration))
		log.Print("Default extension request until ", suggestedEnd.UTC().Format(time.RFC3339))

		// if the 'end' request parameter is set
	} else {
		var err error
		suggestedEnd, err = time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		log.Print("Explicit extension request until ", suggestedEnd.UTC().Format(time.RFC3339))
	}

	// check the suggested end date vs the max end date (which is already set in our implementation)
	//log.Print("Potential rights end = ", licenseStatus.PotentialRights.End.UTC().Format(time.RFC3339))
	if suggestedEnd.After(*licenseStatus.PotentialRights.End) {
		msg := "Attempt to extend with a date greater than max end = " + licenseStatus.PotentialRights.End.UTC().Format(time.RFC3339)
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: msg}, http.StatusForbidden)
		return
	}
	// check the suggested end date vs the current end date
	if suggestedEnd.Before(currentEnd) {
		msg := "Attempt to extend with a date before the current end date"
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: msg}, http.StatusForbidden)
		return
	}

	// add a log
	logging.Print("License extended until " + suggestedEnd.UTC().Format(time.RFC3339))

	// create a renew event with a static device name
	event := makeEvent(status.EVENT_RENEWED, "subscription", "suscription", licenseStatus.ID)
	err = s.Transactions().Add(*event, status.EVENT_RENEWED_INT)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// update a license via a call to the lcp Server
	var httpStatusCode int
	httpStatusCode, err = updateLicense(suggestedEnd, licenseID)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("LCP license PATCH returned HTTP error code " + strconv.Itoa(httpStatusCode))

		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: err.Error()}, httpStatusCode)
		return
	}
	// update the license status fields; the status is active
	licenseStatus.Status = status.STATUS_ACTIVE
	licenseStatus.CurrentEndLicense = &suggestedEnd
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	log.Print("Update timestamp ", event.Timestamp.UTC().Format(time.RFC3339))

	// update the license status in db
	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// fill the localized 'message', the 'links' and 'event' objects in the license status
	err = fillLicenseStatus(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// return the updated license status to the caller
	// the device count must not be sent in json to the caller
	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// FilterLicenseStatuses returns a sequence of license statuses, in their id order
// function for detecting licenses which used a lot of devices
func FilterLicenseStatuses(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_JSON)

	// Get request parameters. If not defined, set default values
	rDevices := r.FormValue("devices")
	if rDevices == "" {
		rDevices = "0"
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
		problem.Error(w, r, problem.Problem{Type: problem.FILTER_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	page, err := strconv.ParseInt(rPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.FILTER_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	perPage, err := strconv.ParseInt(rPerPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.FILTER_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if (page < 1) || (perPage < 1) || (devicesLimit < 0) {
		problem.Error(w, r, problem.Problem{Type: problem.FILTER_BAD_REQUEST, Detail: "Devices, page, per_page must be positive number"}, http.StatusBadRequest)
		return
	}

	page--

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
func ListRegisteredDevices(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_JSON)

	vars := mux.Vars(r)
	licenseID := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	registeredDevicesList := transactions.RegisteredDevicesList{Devices: make([]transactions.Device, 0), ID: licenseStatus.LicenseRef}

	fn := s.Transactions().ListRegisteredDevices(licenseStatus.ID)
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

// LendingCancellation cancels (before use) or revokes (after use)  a license.
// parameters:
//
//	key: license id
//	partial license status: the new status and a message indicating why the status is being changed
//	The new status can be either STATUS_CANCELLED or STATUS_REVOKED
func LendingCancellation(w http.ResponseWriter, r *http.Request, s Server) {
	// get the license id
	vars := mux.Vars(r)
	licenseID := vars["key"]

	logging.Print("Revoke or Cancel the License " + licenseID)

	// get the current license status
	licenseStatus, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		// erroneous license id
		if licenseStatus == nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			return
		}
		// other error
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// get the partial license status document
	var newStatus licensestatuses.LicenseStatus
	err = decodeJsonLicenseStatus(r, &newStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// the new status must be either cancelled or revoked
	if newStatus.Status != status.STATUS_REVOKED && newStatus.Status != status.STATUS_CANCELLED {
		msg := "The new status must be either cancelled or revoked"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// cancelling is only possible when the status is ready
	if newStatus.Status == status.STATUS_CANCELLED && licenseStatus.Status != status.STATUS_READY {
		msg := "The license is not on ready state, it can't be cancelled"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}
	// revocation is only possible when the status is ready or active
	if newStatus.Status == status.STATUS_REVOKED && licenseStatus.Status != status.STATUS_READY && licenseStatus.Status != status.STATUS_ACTIVE {
		msg := "The license is not on ready or active state, it can't be revoked"
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: msg}, http.StatusBadRequest)
		return
	}

	// override the new status, revoked -> cancelled, if the current status is ready
	if newStatus.Status == status.STATUS_REVOKED && licenseStatus.Status == status.STATUS_READY {
		newStatus.Status = status.STATUS_CANCELLED
	}

	// the new expiration time is now
	currentTime := time.Now().UTC().Truncate(time.Second)

	// update the license with the new expiration time, via a call to the lcp Server
	httpStatusCode, erru := updateLicense(currentTime, licenseID)
	if erru != nil {
		problem.Error(w, r, problem.Problem{Detail: erru.Error()}, http.StatusInternalServerError)
		return
	}
	if httpStatusCode != http.StatusOK && httpStatusCode != http.StatusPartialContent { // 200, 206
		err = errors.New("License update notif to lcp server failed with http code " + strconv.Itoa(httpStatusCode))
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, httpStatusCode)
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
	deviceID := "system"
	event := makeEvent(st, deviceName, deviceID, licenseStatus.ID)
	err = s.Transactions().Add(*event, ty)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// update the license status properties with the new status & expiration item (now)
	// the potential end timestamp is also removed.
	licenseStatus.Status = newStatus.Status
	licenseStatus.CurrentEndLicense = &currentTime
	licenseStatus.Updated.Status = &currentTime
	licenseStatus.Updated.License = &currentTime
	licenseStatus.PotentialRights = nil

	// update the license status in db
	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// makeLicenseStatus sets fields of license status according to the config file
// and creates needed inner objects of license status
func makeLicenseStatus(license license.License, ls *licensestatuses.LicenseStatus) {
	ls.LicenseRef = license.ID

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
func getEvents(ls *licensestatuses.LicenseStatus, s Server) error {
	events := make([]transactions.Event, 0)

	fn := s.Transactions().GetByLicenseStatusId(ls.ID)
	var err error
	var event transactions.Event
	for event, err = fn(); err == nil; event, err = fn() {
		events = append(events, event)
	}

	if err == transactions.ErrNotFound {
		ls.Events = events
		err = nil
	}

	return err
}

// makeLinks creates and adds links to the license status
func makeLinks(ls *licensestatuses.LicenseStatus) {
	lsdBaseURL := config.Config.LsdServer.PublicBaseUrl
	licenseLinkURL := config.Config.LsdServer.LicenseLinkUrl
	lcpBaseURL := config.Config.LcpServer.PublicBaseUrl

	usableLicense := (ls.Status == status.STATUS_READY || ls.Status == status.STATUS_ACTIVE)
	registerAvailable := config.Config.LicenseStatus.Register && usableLicense
	licenseHasRightsEnd := ls.CurrentEndLicense != nil && !(*ls.CurrentEndLicense).IsZero()
	returnAvailable := config.Config.LicenseStatus.Return && licenseHasRightsEnd && usableLicense

	// add the possibility to renew an expired license
	var renewableLicense bool
	if config.Config.LicenseStatus.RenewExpired {
		renewableLicense = usableLicense || ls.Status == status.STATUS_EXPIRED
	}
	renewAvailable := config.Config.LicenseStatus.Renew && licenseHasRightsEnd && renewableLicense
	renewPageUrl := config.Config.LicenseStatus.RenewPageUrl
	renewCustomUrl := config.Config.LicenseStatus.RenewCustomUrl

	links := new([]licensestatuses.Link)

	// add a self link (required by ODL)
	link := licensestatuses.Link{Href: lsdBaseURL + "/licenses/" + ls.LicenseRef + "/status", Rel: "self", Type: api.ContentType_LSD_JSON, Templated: false}
	*links = append(*links, link)

	// if the link template to the license is set
	if licenseLinkURL != "" {
		licenseLinkURLFinal := expandUriTemplate(licenseLinkURL, "license_id", ls.LicenseRef)
		link := licensestatuses.Link{Href: licenseLinkURLFinal, Rel: "license", Type: api.ContentType_LCP_JSON, Templated: false}
		*links = append(*links, link)
		// default template
	} else {
		link := licensestatuses.Link{Href: lcpBaseURL + "/api/v1/licenses/" + ls.LicenseRef, Rel: "license", Type: api.ContentType_LCP_JSON, Templated: false}
		*links = append(*links, link)
	}
	// if register is set
	if registerAvailable {
		link := licensestatuses.Link{Href: lsdBaseURL + "/licenses/" + ls.LicenseRef + "/register{?id,name}", Rel: "register", Type: api.ContentType_LSD_JSON, Templated: true}
		*links = append(*links, link)
	}
	// if return is set
	if returnAvailable {
		link := licensestatuses.Link{Href: lsdBaseURL + "/licenses/" + ls.LicenseRef + "/return{?id,name}", Rel: "return", Type: api.ContentType_LSD_JSON, Templated: true}
		*links = append(*links, link)
	}

	// if renew is set
	if renewAvailable {
		var link licensestatuses.Link
		if renewPageUrl != "" {
			// renewal is managed via a web page
			expandedUrl := expandUriTemplate(renewPageUrl, "license_id", ls.LicenseRef)
			link = licensestatuses.Link{Href: expandedUrl, Rel: "renew", Type: api.ContentType_TEXT_HTML}
		} else if renewCustomUrl != "" {
			// renewal is managed via a specific service handled by the provider.
			// The expanded renew url is itself a templated Url, which may of may not contain query parameters.
			// Warning: {&end,id,name} (note the '&') may not be properly processed by most clients.
			expandedUrl := expandUriTemplate(renewCustomUrl, "license_id", ls.LicenseRef)
			if strings.Contains(renewCustomUrl, "?") {
				expandedUrl = expandedUrl + "{&end,id,name}"
			} else {
				expandedUrl = expandedUrl + "{?end,id,name}"
			}
			link = licensestatuses.Link{Href: expandedUrl, Rel: "renew", Type: api.ContentType_LSD_JSON, Templated: true}
		} else {
			// this is the most usual case, i.e. a simple renew link
			link = licensestatuses.Link{Href: lsdBaseURL + "/licenses/" + ls.LicenseRef + "/renew{?end,id,name}", Rel: "renew", Type: api.ContentType_LSD_JSON, Templated: true}
		}
		*links = append(*links, link)
	}

	ls.Links = *links
}

// expandUriTemplate resolves a url template from the configuration to a url the system can embed in a status document
func expandUriTemplate(uriTemplate, variable, value string) string {
	template, _ := uritemplates.Parse(uriTemplate)
	values := make(map[string]interface{})
	values[variable] = value
	expanded, err := template.Expand(values)
	if err != nil {
		log.Printf("failed to expand an uri template: %s", uriTemplate)
		return uriTemplate
	}
	return expanded
}

// makeEvent creates an event and fill it
func makeEvent(status string, deviceName string, deviceID string, licenseStatusFk int) *transactions.Event {
	event := transactions.Event{}
	event.DeviceId = deviceID
	event.DeviceName = deviceName
	event.Timestamp = time.Now().UTC().Truncate(time.Second)
	event.Type = status
	event.LicenseStatusFk = licenseStatusFk

	return &event
}

// decodeJsonLicenseStatus decodes license status json to the object
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
func updateLicense(timeEnd time.Time, licenseID string) (int, error) {
	// get the lcp server url
	lcpBaseURL := config.Config.LcpServer.PublicBaseUrl
	if len(lcpBaseURL) <= 0 {
		return 0, errors.New("undefined Config.LcpServer.PublicBaseUrl")
	}
	// create a minimum license object, limited to the license id plus rights
	// FIXME: remove the id (here and in the lcpserver license.go)
	minLicense := license.License{ID: licenseID, Rights: new(license.UserRights)}
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
	lcpURL := lcpBaseURL + "/licenses/" + licenseID
	// message to the console
	//log.Println("PATCH " + lcpURL)
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

	log.Println("Error Notifying Lcp Server of License update (" + licenseID + "):" + err.Error())
	return 0, err
}

// fillLicenseStatus fills the 'message' field, the 'links' and 'event' objects in the license status
func fillLicenseStatus(ls *licensestatuses.LicenseStatus, s Server) error {
	// add the message
	ls.Message = "The license is in " + ls.Status + " state"
	// add the links
	makeLinks(ls)
	// add the events
	err := getEvents(ls, s)

	return err
}

// Ping is a simple health check
func Ping(w http.ResponseWriter, r *http.Request, s Server) {
	w.WriteHeader(http.StatusOK)
}
