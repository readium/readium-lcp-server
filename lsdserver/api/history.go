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

	if licenseStatus.PotentialRights.End != nil {
		diff := currentDateTime.Sub(*(licenseStatus.PotentialRights.End))

		if (!licenseStatus.PotentialRights.End.IsZero()) && (diff > 0) && ((licenseStatus.Status == status.STATUS_ACTIVE) || (licenseStatus.Status == status.STATUS_READY)) {
			licenseStatus.Status = status.STATUS_EXPIRED
			s.History().Update(*licenseStatus)
		}
	}

	err = fillLicenseStatus(licenseStatus, r, s)
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
	w.Header().Set("Content-Type", "application/vnd.readium.license.status.v1.0+json")
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
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/registration", Detail: "parameters mandatory"}, http.StatusBadRequest)
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

	if licenseStatus.Status == status.STATUS_READY {
		licenseStatus.Status = status.STATUS_ACTIVE
		licenseStatus.Updated.License = &event.Timestamp

	}

	licenseStatus.DeviceCount += 1

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	err = fillLicenseStatus(licenseStatus, r, s)
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

func LendingReturn(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", "application/vnd.readium.license.status.v1.0+json")
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
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/already", Detail: "Already returned"}, http.StatusForbidden)
		return
	case status.STATUS_EXPIRED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return/expired", Detail: "Already expired"}, http.StatusForbidden)
		return
	case status.STATUS_ACTIVE:
		licenseStatus.Status = status.STATUS_RETURNED
		break
	case status.STATUS_READY:
		licenseStatus.Status = status.STATUS_CANCELLED
		break
	case status.STATUS_CANCELLED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: "Already returned"}, http.StatusBadRequest)
		return
	case status.STATUS_REVOKED:
		problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: "Already returned"}, http.StatusBadRequest)
		return
	}

	//check if device is activated
	if deviceId != "" {
		deviceStatus, err := s.Transactions().CheckDeviceStatus(licenseStatus.Id, deviceId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		if deviceStatus == status.TYPE_RETURN || deviceStatus == "" {
			problem.Error(w, r, problem.Problem{Type: "http://readium.org/license-status-document/error/return", Detail: "Not activated"}, http.StatusBadRequest)
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

	go updateLicense(event.Timestamp, licenseFk)

	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(licenseStatus)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

func LendingRenewal(w http.ResponseWriter, r *http.Request, s Server) {
	w.Header().Set("Content-Type", "application/vnd.readium.license.status.v1.0+json")
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

	timeEndString := r.FormValue("end")
	if timeEndString == "" {
		renewDays := config.Config.LicenseStatus.RenewDays
		if renewDays == 0 {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: "renew_days not found"}, http.StatusInternalServerError)
			return
		}
		if licenseStatus.PotentialRights.End == nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: "potential rights not set"}, http.StatusInternalServerError)
			return
		}
		end := licenseStatus.PotentialRights.End.Add(time.Hour * 24 * time.Duration(renewDays))
		licenseStatus.PotentialRights.End = &end
	} else {
		expirationEnd, err := time.Parse(time.RFC3339, timeEndString)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		licenseStatus.PotentialRights.End = &expirationEnd
	}

	licenseStatus.Updated.Status = &event.Timestamp
	licenseStatus.Updated.License = &event.Timestamp
	licenseStatus.Status = status.STATUS_ACTIVE

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	/*lcp update rights*/
	go updateLicense(event.Timestamp, licenseFk)

	err = fillLicenseStatus(licenseStatus, r, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

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
	var parsedLs history.LicenseStatus
	err = decodeJsonLicenseStatus(r, &parsedLs)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	licenseStatus.Status = parsedLs.Status
	currentTime := time.Now()

	licenseStatus.Updated.Status = &currentTime
	licenseStatus.Updated.License = &currentTime

	err = s.History().Update(*licenseStatus)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: SERVER_INTERNAL_ERROR, Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	go updateLicense(currentTime, licenseFk)
}

func makeLicenseStatus(license license.License, ls *history.LicenseStatus) {
	ls.LicenseRef = license.Id

	registerAvailable := config.Config.LicenseStatus.Register
	rentingDays := config.Config.LicenseStatus.RentingDays

	ls.PotentialRights = new(history.PotentialRights)

	if rentingDays != 0 {
		end := license.Issued.Add(time.Hour * 24 * time.Duration(rentingDays))
		ls.PotentialRights.End = &end
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

func decodeJsonLicenseStatus(r *http.Request, ls *history.LicenseStatus) error {
	var dec *json.Decoder

	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == "application/x-www-form-urlencoded" {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	err := dec.Decode(&ls)

	return err
}

func updateLicense(timeEnd time.Time, licenseRef string) {
	l := license.License{Id: licenseRef, Rights: new(license.UserRights)}
	l.Rights.End = &timeEnd

	lcpBaseUrl := config.Config.LcpBaseUrl
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
		req.Header.Add("Content-Type", "application/vnd.readium.lcp.license.1-0+json")
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

func fillLicenseStatus(ls *history.LicenseStatus, r *http.Request, s Server) error {
	makeLinks(ls)

	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &ls.Message, ls.Status)

	err := getEvents(ls, s)

	return err
}
