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

//without privacy for now
//make privacy
//return loc errors
func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	var lic license.License

	err := apilcp.DecodeJsonLicense(r, &lic)

	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
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

	//prepare links
	makeLinks(licenseStatus)

	//localize message
	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &licenseStatus.Message, licenseStatus.Status)

	//get events
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

	/*TODO: check for constraints? what constrains?*/
	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	if licenseStatus.Status == status.STATUS_REVOKED {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/registration", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	event := makeEvent(status.STATUS_ACTIVE, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Status = event.Type

	//when register ++1 to device count
	licenseStatus.DeviceCount += 1

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	//do we need to return a license status here?
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

	/*TODO: need to check constraints here?*/
	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	if licenseStatus.Status == status.STATUS_RETURNED {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/already", Detail: err.Error()}, http.StatusForbidden)
		return
	}
	if licenseStatus.Status == status.STATUS_EXPIRED {
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/expired", Detail: err.Error()}, http.StatusForbidden)
		return
	}

	//check if device is active in this way?
	deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	if deviceStatus != status.STATUS_ACTIVE {
		/*what error to return here?*/
	}

	event := makeEvent(status.STATUS_RETURNED, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp

	//from spec about return
	if licenseStatus.Status == status.STATUS_READY {
		licenseStatus.Status = status.STATUS_CANCELLED
	}
	if licenseStatus.Status == status.STATUS_ACTIVE {
		licenseStatus.Status = status.STATUS_RETURNED
	}

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	/*TODO lcp update rights*/

	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(licenseStatus)
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

	/*TODO: need to check constraints here?*/
	deviceId := r.FormValue("device_id")
	deviceName := r.FormValue("device_name")

	//check if device is active in this way?
	deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	if deviceStatus != status.STATUS_ACTIVE {
		/*what error to return here?*/
	}

	event := makeEvent(status.STATUS_RETURNED, deviceName, deviceId, licenseStatus.Id)

	err = s.Transactions().Add(*event)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	licenseStatus.Status = event.Type

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	/*TODO lcp update rights*/

	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(licenseStatus)
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
		makeLinks(&it)

		acceptLanguages := r.Header.Get("Accept-Language")
		localization.LocalizeMessage(acceptLanguages, &it.Message, it.Status)

		err = getEvents(&it, s)
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
	//profile info?
	lsdBaseUrl := config.Config.LsdBaseUrl
	registerAvailable := config.Config.LicenseStatus.Register
	returnAvailable := config.Config.LicenseStatus.Return
	renewAvailable := config.Config.LicenseStatus.Renew

	ls.Links = make(map[string][]history.Link)
	ls.Links["license"] = make([]history.Link, 1)
	ls.Links["license"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "",
		"application/vnd.readium.lcp.license.v1.0+json", "", false)

	if registerAvailable {
		ls.Links["register"] = make([]history.Link, 1)
		ls.Links["register"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/register{?id,name}",
			"application/vnd.readium.license.status.v1.0+json", "", true)
	}

	if returnAvailable {
		ls.Links["return"] = make([]history.Link, 1)
		ls.Links["return"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/return{?id,name}",
			"application/vnd.readium.lcp.license-1.0+json", "", true)
	}

	if renewAvailable {
		ls.Links["renew"] = make([]history.Link, 2)
		ls.Links["renew"][0] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew",
			"text/html", "", false)
		ls.Links["renew"][1] = createLink(lsdBaseUrl, ls.LicenseRef, "/renew{?end,id,name}",
			"application/vnd.readium.lcp.license-1.0+json", "", true)
	}
}

func createLink(publicBaseUrl string, licenseRef string, page string,
	typeLink string, title string, templated bool) history.Link {
	var link history.Link
	link.Href = publicBaseUrl + "/license/" + licenseRef + page
	link.Title = title
	link.Type = typeLink

	link.Templated = templated

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
