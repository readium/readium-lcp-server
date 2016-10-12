package apilsd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/epub"
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
}

//CreateLicenseStatusDocument create license status and add it to database
func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	var lic license.License
	err := apilcp.DecodeJsonLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	var ls licensestatuses.LicenseStatus
	makeLicenseStatus(lic, &ls)

	err = s.LicenseStatuses().Add(ls)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

//GetLicenseStatusDocument get license status from database by licese id
//checks potential_rights_end and fill it
func GetLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile("error", http.StatusNotFound, logging.BASIC_FUNCTION)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.BASIC_FUNCTION)
		return
	}

	currentDateTime := time.Now()

	if licenseStatus.PotentialRights != nil {
		diff := currentDateTime.Sub(*(licenseStatus.PotentialRights.End))

		if (!licenseStatus.PotentialRights.End.IsZero()) && (diff > 0) && ((licenseStatus.Status == status.STATUS_ACTIVE) || (licenseStatus.Status == status.STATUS_READY)) {
			licenseStatus.Status = status.STATUS_EXPIRED
			s.LicenseStatuses().Update(*licenseStatus)
		}
	}

	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.BASIC_FUNCTION)
		return
	}

	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)

	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.BASIC_FUNCTION)
		return
	}

	logging.WriteToFile("sucsess", http.StatusOK, logging.BASIC_FUNCTION)
}

//RegisterDevice register device using device_id & device_name request parameters
//& returns updated and filled license status
func RegisterDevice(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile("error", http.StatusNotFound, logging.SUCCESS_REGISTRATION)
			logging.WriteToFile("error", http.StatusNotFound, logging.REJECT_REGISTRATION)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	dILen := len(deviceId)
	dNLen := len(deviceName)

	//check mandatory request parameters
	if (dILen == 0) || (dILen > 255) || (dNLen == 0) || (dNLen > 255) {
		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: "device_id and device_name are mandatory and maximum lenght is 255 symbols "}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_REGISTRATION)
		return
	}

	//check status of license status
	if (licenseStatus.Status != status.STATUS_ACTIVE) && (licenseStatus.Status != status.STATUS_READY) {
		problem.Error(w, r, problem.Problem{Type: problem.REGISTRATION_BAD_REQUEST, Detail: "License is not active"}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_REGISTRATION)
		return
	}

	//check the existence of device in license status
	deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}
	if deviceStatus != "" {
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: "Device has been already registered"}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_REGISTRATION)
		return
	}

	//make event for register transaction
	event := makeEvent(status.TYPE_REGISTER, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 1)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp

	//check & set the status of the license status
	if licenseStatus.Status == status.STATUS_READY {
		licenseStatus.Status = status.STATUS_ACTIVE
		licenseStatus.Updated.License = &event.Timestamp

	}

	*licenseStatus.DeviceCount += 1

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}

	//fill license status
	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}

	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_REGISTRATION)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_REGISTRATION)
		return
	}
	logging.WriteToFile("success", http.StatusOK, logging.SUCCESS_REGISTRATION)
}

//LendingReturn checks that the calling device is activated, then modifies
//the end date associated with the given license & returns updated and filled license status
func LendingReturn(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile("error", http.StatusNotFound, logging.SUCCESS_RETURN)
			logging.WriteToFile("error", http.StatusNotFound, logging.REJECT_RETURN)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	//checks request parameters
	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RETURN)
		return
	}

	//check & set the status of license status according to its current value
	switch licenseStatus.Status {
	case status.STATUS_RETURNED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/already", Detail: "License has been already returned"}, http.StatusForbidden)
		logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RETURN)
		return
	case status.STATUS_EXPIRED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/expired", Detail: "License is expired"}, http.StatusForbidden)
		logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RETURN)
		return
	case status.STATUS_ACTIVE:
		licenseStatus.Status = status.STATUS_RETURNED
		break
	case status.STATUS_READY:
		licenseStatus.Status = status.STATUS_CANCELLED
		break
	case status.STATUS_CANCELLED:
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: "License is cancelled"}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RETURN)
		return
	case status.STATUS_REVOKED:
		problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: "License is revoked"}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RETURN)
		return
	}

	//check if device is activated
	if deviceId != "" {
		deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
			logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
			return
		}
		if deviceStatus == status.TYPE_RETURN || deviceStatus == "" {
			problem.Error(w, r, problem.Problem{Type: problem.RETURN_BAD_REQUEST, Detail: "Device is not activated"}, http.StatusBadRequest)
			logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_RETURN)
			return
		}
	}

	//create event for lending return
	event := makeEvent(status.TYPE_RETURN, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 2)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
		return
	}

	//update licenseStatus
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
		return
	}

	//update license using LCP Server
	go updateLicense(event.Timestamp, licenseFk)

	//fill license status
	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
		return
	}

	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RETURN)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RETURN)
		return
	}
	logging.WriteToFile("success", http.StatusOK, logging.SUCCESS_RETURN)
}

//LendingRenewal checks that the calling device is activated, then modifies
//the end date associated with the license & returns updated and filled license status
func LendingRenewal(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_LSD_JSON)
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			logging.WriteToFile("error", http.StatusNotFound, logging.SUCCESS_RENEW)
			logging.WriteToFile("error", http.StatusNotFound, logging.REJECT_RENEW)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	//check the request parameters
	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: err.Error()}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_RENEW)
		return
	}

	if (licenseStatus.Status != status.STATUS_ACTIVE) && (licenseStatus.Status != status.STATUS_READY) {
		problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: "License is not active"}, http.StatusBadRequest)
		logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_RENEW)
		return
	}

	//check if device is active
	if deviceId != "" {
		deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
			logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
			return
		}
		if deviceStatus != status.TYPE_REGISTER {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_BAD_REQUEST, Detail: "The device is not active for this license"}, http.StatusBadRequest)
			logging.WriteToFile("success", http.StatusBadRequest, logging.REJECT_RENEW)
			return
		}
	}

	//set new date for potential_rights_end
	//if request parameter 'end' is empty, it used renew_days parameter from config
	timeEndString := r.FormValue("end")
	if timeEndString == "" {
		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: "renew_days not found"}, http.StatusInternalServerError)
			logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RENEW)
			return
		}
		if licenseStatus.PotentialRights == nil {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: "potential rights not set"}, http.StatusInternalServerError)
			logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RENEW)
			return
		}
		end := licenseStatus.PotentialRights.End.Add(time.Hour * 24 * time.Duration(renewDays))
		licenseStatus.PotentialRights.End = &end
	} else {
		expirationEnd, err := time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: problem.RENEW_REJECT, Detail: err.Error()}, http.StatusInternalServerError)
			logging.WriteToFile("success", http.StatusForbidden, logging.REJECT_RENEW)
			return
		}
		licenseStatus.PotentialRights.End = &expirationEnd
	}
	event := makeEvent(status.TYPE_RENEW, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 3)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
		return
	}

	//update license status fields
	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	licenseStatus.Status = status.STATUS_ACTIVE

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
		return
	}

	//update license using LCP Server
	go updateLicense(event.Timestamp, licenseFk)

	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
		return
	}

	licenseStatus.DeviceCount = nil
	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.SUCCESS_RENEW)
		logging.WriteToFile("error", http.StatusInternalServerError, logging.REJECT_RENEW)
		return
	}

	logging.WriteToFile("success", http.StatusOK, logging.SUCCESS_RENEW)
}

//FilterLicenseStatuses returns a sequence of license statuses, in their id order
//function for detecting licenses which used a lot of devices
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
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	page, err := strconv.ParseInt(rPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	perPage, err := strconv.ParseInt(rPerPage, 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if (page < 1) || (perPage < 1) || (devicesLimit < 1) {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "devices, page, per_page must be positive number"}, http.StatusBadRequest)
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
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//ListRegisteredDevices returns data about the use of a given license
func ListRegisteredDevices(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", api.ContentType_JSON)

	vars := mux.Vars(r)
	licenseFk := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
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
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//CancelLicenseStatus cancel or revoke (according to the status) a license
func CancelLicenseStatus(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	licenseFk := vars["key"]

	licenseStatus, err := s.LicenseStatuses().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	if licenseStatus.Status != status.STATUS_READY {
		problem.Error(w, r, problem.Problem{Type: problem.CANCEL_BAD_REQUEST, Detail: "The new status is not compatible with current status"}, http.StatusBadRequest)
	}

	var parsedLs licensestatuses.LicenseStatus
	err = decodeJsonLicenseStatus(r, &parsedLs)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Status = parsedLs.Status

	currentTime := time.Now()

	licenseStatus.Updated.Status = &currentTime
	licenseStatus.Updated.License = &currentTime

	err = s.LicenseStatuses().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: problem.SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	//update license using LCP Server
	go updateLicense(currentTime, licenseFk)
}

//makeLicenseStatus sets fields of license status according to the config file
//and creates needed inner objects of license status
func makeLicenseStatus(license license.License, ls *licensestatuses.LicenseStatus) {
	ls.LicenseRef = license.Id

	registerAvailable := config.Config.LicenseStatus.Register
	rentingDays := config.Config.LicenseStatus.RentingDays

	ls.PotentialRights = new(licensestatuses.PotentialRights)
	if rentingDays != 0 {
		end := license.Issued.Add(time.Hour * 24 * time.Duration(rentingDays))
		ls.PotentialRights.End = &end
	}

	if registerAvailable {
		ls.Status = status.STATUS_READY
	} else {
		ls.Status = status.STATUS_ACTIVE
	}

	ls.Updated = new(licensestatuses.Updated)
	ls.Updated.License = &license.Issued

	currentTime := time.Now()
	ls.Updated.Status = &currentTime

	count := 0
	ls.DeviceCount = &count
}

//getEvents gets the events from database for the license status
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

//makeLinks creates and adds links to the license status
func makeLinks(ls *licensestatuses.LicenseStatus) {
	lsdBaseUrl := config.Config.LsdServer.PublicBaseUrl
	lcpBaseUrl := config.Config.LcpServer.PublicBaseUrl
	registerAvailable := config.Config.LicenseStatus.Register
	returnAvailable := config.Config.LicenseStatus.Return
	renewAvailable := config.Config.LicenseStatus.Renew

	ls.Links = make(map[string][]licensestatuses.Link)
	ls.Links["license"] = make([]licensestatuses.Link, 1)
	ls.Links["license"][0] = createLink(lcpBaseUrl, ls.LicenseRef, "",
		api.ContentType_LCP_JSON, false)

	if registerAvailable {
		ls.Links["register"] = make([]licensestatuses.Link, 1)
		ls.Links["register"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/register{?id,name}",
			api.ContentType_LSD_JSON, true)
	}

	if returnAvailable {
		ls.Links["return"] = make([]licensestatuses.Link, 1)
		ls.Links["return"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/return{?id,name}",
			api.ContentType_LCP_JSON, true)
	}

	if renewAvailable {
		ls.Links["renew"] = make([]licensestatuses.Link, 2)
		ls.Links["renew"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew", epub.ContentType_HTML, false)
		ls.Links["renew"][1] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew{?end,id,name}",
			api.ContentType_LCP_JSON, true)
	}
}

//createLink creates a link and fills it
func createLink(publicBaseUrl string, licenseRef string, page string,
	typeLink string, templated bool) licensestatuses.Link {
	link := licensestatuses.Link{Href: publicBaseUrl + "/licenses/" + licenseRef + page,
		Type: typeLink, Templated: templated}
	return link
}

//makeEvent creates an event and fill it
func makeEvent(status string, deviceName string, deviceId string, licenseStatusFk int) *transactions.Event {
	event := transactions.Event{}
	event.DeviceId = deviceId
	event.DeviceName = deviceName
	event.Timestamp = time.Now()
	event.Type = status
	event.LicenseStatusFk = licenseStatusFk

	return &event
}

//decodeJsonLicenseStatus decodes license status json to the object
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

//updateLicense updates license using LCP Server
func updateLicense(timeEnd time.Time, licenseRef string) {
	l := license.License{Id: licenseRef, Rights: new(license.UserRights)}
	l.Rights.End = &timeEnd

	lcpBaseUrl := config.Config.LcpServer.PublicBaseUrl
	if len(lcpBaseUrl) > 0 {
		var lcpClient = &http.Client{
			Timeout: time.Second * 10,
		}
		pr, pw := io.Pipe()
		go func() {
			_ = json.NewEncoder(pw).Encode(l)
			pw.Close()
		}()
		req, err := http.NewRequest("PATCH", lcpBaseUrl+"/licenses/"+l.Id, pr)
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			log.Println("Error Notify Lcp Server of License (" + l.Id + "):" + err.Error())
		} else {
			if response.StatusCode != http.StatusOK {
				log.Println("Notify Lcp Server of License (" + l.Id + ") = " + strconv.Itoa(response.StatusCode))
			}
		}
	}
}

//fillLicenseStatus fills object 'links' and field 'message' in license status
func fillLicenseStatus(ls *licensestatuses.LicenseStatus, r *http.Request, s Server) error {
	makeLinks(ls)

	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &ls.Message, ls.Status)

	err := getEvents(ls, s)

	return err
}
