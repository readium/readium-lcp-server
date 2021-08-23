// Copyright 2021 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilsd

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
)

// UserData represents the payload requested to the CMS.
// This is a simplified version of a partial license, easy to generate for any CMS developer.
// PassphraseHash is the result of the hash calculation, as an hex-encoded string
type UserData struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	PassphraseHash string `json:"passphrasehash"`
	Hint           string `json:"hint"`
}

const Sha256_URL string = "http://www.w3.org/2001/04/xmlenc#sha256"

// GetFreshLicense gets a fresh license from the License Server
// after requesting user data (for this license) from the CMS.
func GetFreshLicense(w http.ResponseWriter, r *http.Request, s Server) {

	// get the licenseID parameter
	vars := mux.Vars(r)
	licenseID := vars["key"]

	// check if the license is known from the Status Document Server
	statusDoc, err := s.LicenseStatuses().GetByLicenseID(licenseID)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
	}

	// get a fresh license from the License Server
	freshLicense, err := getLicense(licenseID)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// return the fresh license to the caller
	w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Set("Content-Disposition", "attachment; filename=\"license.lcpl\"")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	err = enc.Encode(freshLicense)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	log.Println("Get license / id " + licenseID + " / status " + statusDoc.Status)
}

// GetLicense gets a fresh license from the License Server
func getLicense(licenseID string) (lic license.License, err error) {

	// get user data from the CMS
	var userData UserData
	userData, err = getUserData(licenseID)
	if err != nil {
		return
	}

	// init the partial license to be sent to the License Server
	var plic license.License
	plic, err = initPartialLicense(licenseID, userData)
	if err != nil {
		return
	}

	// fetch the license from the License Server
	lic, err = fetchLicense(plic)
	if err != nil {
		return
	}
	return
}

// getUserData gets user data from the CMS, as a partial license
func getUserData(licenseID string) (userData UserData, err error) {

	// get the url of the CMS
	userURL := strings.Replace(config.Config.LsdServer.UserDataUrl, "{license_id}", licenseID, -1)
	if userURL == "" {
		err = errors.New("Get User Data from License ID: UserDataUrl missing from the server configuration")
		return
	}

	// fetch user data
	client := &http.Client{}
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return
	}
	auth := config.Config.CMSAccessAuth
	if auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if resp.StatusCode != 200 {
		var errStatus problem.Problem
		err = dec.Decode(&errStatus)
		if err != nil {
			return
		}
		err = errors.New("Get User Data from License ID: " + errStatus.Title + " - " + errStatus.Detail)
		return
	} else {
		// decode user data
		err = dec.Decode(&userData)
		if err != nil {
			return
		}
	}

	return
}

// initPartialLicense inits the partial license to be sent to the License Server
func initPartialLicense(licenseID string, userData UserData) (plic license.License, err error) {
	plic.ID = licenseID
	plic.User.ID = userData.ID
	plic.User.Name = userData.Name
	plic.User.Email = userData.Email
	// we decide that name and email will be encrypted
	encryptedAttrs := []string{"name", "email"}
	plic.User.Encrypted = encryptedAttrs

	plic.Encryption.UserKey.Algorithm = Sha256_URL
	plic.Encryption.UserKey.Hint = userData.Hint
	plic.Encryption.UserKey.HexValue = userData.PassphraseHash
	return
}

// fetchLicense fetches a license from the License Server
func fetchLicense(plic license.License) (lic license.License, err error) {
	// json encode the partial license
	jplic, err := json.Marshal(plic)
	if err != nil {
		return
	}

	// send the partial license to the License Server and get back a fresh license
	licenseUrl := config.Config.LcpServer.PublicBaseUrl + "/licenses/" + plic.ID
	client := &http.Client{}
	req, err := http.NewRequest("POST", licenseUrl, bytes.NewReader(jplic))
	if err != nil {
		return
	}
	auth := config.Config.LcpUpdateAuth
	if auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if resp.StatusCode != 200 {
		var errStatus problem.Problem
		err = dec.Decode(&errStatus)
		if err != nil {
			return
		}
		err = errors.New("Fetch License: " + errStatus.Title + " - " + errStatus.Detail)
		return
	} else {
		// decode the license, init the return value
		err = dec.Decode(&lic)
		if err != nil {
			return
		}
	}

	return
}
