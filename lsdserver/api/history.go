package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/transactions"
)

type Server interface {
	Transactions() transactions.Transactions
	History() history.History
}

//without privacy for now
//make privacy
//return loc errors
func CreateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	var lic license.License

	err := api.DecodeJsonLicense(r, &lic)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ls history.LicenseStatus
	makeLicenseStatus(lic, &ls)

	ls.Updated = new(history.Updated)

	err = s.History().Add(ls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(201)
}

func GenerateLicenseStatusDocument(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenseFk := vars["key"]

	licenseStatus, err := s.History().GetByLicenseId(licenseFk)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//prepare links
	makeLinks(&licenseStatus)

	//localize message
	acceptLanguages := r.Header.Get("Accept-Language")
	localization.LocalizeMessage(acceptLanguages, &licenseStatus.Message, licenseStatus.Status)

	//get events
	err = getEvents(&licenseStatus, s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(licenseStatus)
}

func makeLicenseStatus(license license.License, ls *history.LicenseStatus) {
	ls.LicenseRef = license.Id

	registerAvailable := config.Config.LicenseStatus.Register
	rentingDays := config.Config.LicenseStatus.RentingDays

	if rentingDays != 0 {
		ls.PotentialRights = new(history.PotentialRights)
		ls.PotentialRights.End = license.Issued.Add(time.Hour * 24 * 7 * time.Duration(rentingDays))
	}

	if registerAvailable {
		ls.Status = history.STATUS_READY
	} else {
		ls.Status = history.STATUS_ACTIVE
	}

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

	if !templated {
		link.Templated = true
	}

	return link
}
