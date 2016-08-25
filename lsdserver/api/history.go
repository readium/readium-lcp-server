package apilsd

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/status"
	"github.com/readium/readium-lcp-server/transactions"
)

type Server interface {
	Transactions() transactions.Transactions
	History() history.History
}

const SERVER_INTERNAL_ERROR = "http://readium.org/license-status-document/error/server"

func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	/*TODO privacy*/
	var lic license.License

	err := apilcp.DecodeJsonLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	var ls history.LicenseStatus
	makeLicenseStatus(lic, &ls)

	err = s.History().Add(ls)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func GetLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]

	licenseStatus, err := s.History().GetByLicenseId(licenseFk)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	currentDateTime := time.Now()
	diff := currentDateTime.Sub(licenseStatus.PotentialRights.End)

	if (!licenseStatus.PotentialRights.End.IsZero()) && (diff > 0) && ((licenseStatus.Status == status.STATUS_ACTIVE) || (licenseStatus.Status == status.STATUS_READY)) {
		licenseStatus.Status = status.STATUS_EXPIRED
		s.History().Update(*licenseStatus)
	}

	makeLinks(licenseStatus)

	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &licenseStatus.Message, licenseStatus.Status)

	err = getEvents(licenseStatus, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

func RegisterDevice(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.History().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	dILen := len(deviceId)
	dNLen := len(deviceName)

	if (dILen == 0) || (dILen > 255) || (dNLen == 0) || (dNLen > 255) {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/registration", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if (licenseStatus.Status != status.STATUS_ACTIVE) && (licenseStatus.Status != status.STATUS_READY) {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/registration", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	event := makeEvent(status.TYPE_REGISTER, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 1)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Status = event.Type

	licenseStatus.DeviceCount += 1

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

func LendingReturn(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.History().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	switch licenseStatus.Status {
	case status.STATUS_RETURNED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/already", Detail: err.Error()}, http.StatusForbidden)
		return
	case status.STATUS_EXPIRED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/expired", Detail: err.Error()}, http.StatusForbidden)
		return
	case status.STATUS_ACTIVE:
		licenseStatus.Status = status.STATUS_RETURNED
		break
	case status.STATUS_READY:
		licenseStatus.Status = status.STATUS_CANCELLED
		break
	case status.STATUS_CANCELLED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: err.Error()}, http.StatusBadRequest)
		return
	case status.STATUS_REVOKED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	//check if device is activated
	if deviceId != "" {
		deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		if deviceStatus != status.STATUS_ACTIVE {
			problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	}

	event := makeEvent(status.TYPE_RETURN, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 2)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	/*TODO lcp update rights*/

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

func LendingRenewal(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]
	licenseStatus, err := s.History().GetByLicenseId(licenseFk)

	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	if (len(deviceName) > 255) || (len(deviceId) > 255) {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/renew", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	//check if device is active
	if deviceId != "" {
		deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		if deviceStatus != status.STATUS_ACTIVE {
			problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/renew", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	}

	event := makeEvent(status.TYPE_RENEW, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event, 3)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	expirationEnd, err := time.Parse(time.RFC3339, r.FormValue("end"))
	if expirationEnd.IsZero() {
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}

		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		licenseStatus.PotentialRights.End = licenseStatus.PotentialRights.End.Add(time.Hour * 24 * 7 * time.Duration(renewDays))
	} else {
		licenseStatus.PotentialRights.End = expirationEnd
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	licenseStatus.Status = status.STATUS_ACTIVE

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	/*TODO lcp update rights*/

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func FilterLicenseStatuses(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", "application/json")

	devicesLimit, err := strconv.ParseInt(r.FormValue("devices"), 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	page, err := strconv.ParseInt(r.FormValue("page"), 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	perPage, err := strconv.ParseInt(r.FormValue("per_page"), 10, 32)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if (page < 1) || (perPage < 1) || (devicesLimit < 1) {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "devices, page, per_page must be positive number"}, http.StatusBadRequest)
		return
	}

	page -= 1

	licenseStatuses := make([]history.LicenseStatus, 0)

	fn := s.History().List(devicesLimit, perPage, page*perPage)
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

func ListRegisteredDevices(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	licenseFk := vars["key"]

	licenseStatus, err := s.History().GetByLicenseId(licenseFk)
	if err != nil {
		if licenseStatus == nil {
			problem.NotFoundHandler(w, r)
			return
		}

		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
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

func CancelLicenseStatus(w http.ResponseWriter, r *http.Request, s Server) {

}

func makeLicenseStatus(license license.License, ls *history.LicenseStatus) {
	ls.LicenseRef = license.Id

	registerAvailable := config.Config.LicenseStatus.Register
	rentingDays := config.Config.LicenseStatus.RentingDays

	ls.PotentialRights = new(history.PotentialRights)

	if rentingDays != 0 {
		ls.PotentialRights.End = license.Issued.Add(time.Hour * 24 * 7 * time.Duration(rentingDays))
	}

	if registerAvailable {
		ls.Status = status.STATUS_READY
	} else {
		ls.Status = status.STATUS_ACTIVE
	}

	ls.Updated = new(history.Updated)
}

func getEvents(ls *history.LicenseStatus, s Server) error {
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

func makeLinks(ls *history.LicenseStatus) {
	lsdBaseUrl := config.Config.LsdBaseUrl
	registerAvailable := config.Config.LicenseStatus.Register
	returnAvailable := config.Config.LicenseStatus.Return
	renewAvailable := config.Config.LicenseStatus.Renew

	ls.Links = make(map[string][]history.Link)
	ls.Links["license"] = make([]history.Link, 1)
	ls.Links["license"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "",
		"application/vnd.readium.lcp.license.v1.0+json", false)

	if registerAvailable {
		ls.Links["register"] = make([]history.Link, 1)
		ls.Links["register"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/register{?id,name}",
			"application/vnd.readium.license.status.v1.0+json", true)
	}

	if returnAvailable {
		ls.Links["return"] = make([]history.Link, 1)
		ls.Links["return"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/return{?id,name}",
			"application/vnd.readium.lcp.license-1.0+json", true)
	}

	if renewAvailable {
		ls.Links["renew"] = make([]history.Link, 2)
		ls.Links["renew"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew",
			"text/html", false)
		ls.Links["renew"][1] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew{?end,id,name}",
			"application/vnd.readium.lcp.license-1.0+json", true)
	}
}

func createLink(publicBaseUrl string, licenseRef string, page string,
	typeLink string, templated bool) history.Link {
	link := history.Link{Href: publicBaseUrl + "/license/" + licenseRef + page,
		Type: typeLink, Templated: templated}
	return link
}

func makeEvent(status string, deviceName string, deviceId string, licenseStatusFk int) *transactions.Event {
	event := transactions.Event{}
	event.DeviceId = deviceId
	event.DeviceName = deviceName
	event.Timestamp = time.Now()
	event.Type = status
	event.LicenseStatusFk = licenseStatusFk

	return &event
}
