/// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package staticapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/problem"
)

// GetFilteredLicenses searches licenses activated by more than n devices
//
func GetFilteredLicenses(w http.ResponseWriter, r *http.Request, s IServer) {

	rDevices := r.FormValue("devices")
	log.Println("Licenses used by " + rDevices + " devices")
	if rDevices == "" {
		rDevices = "0"
	}

	if lic, err := s.LicenseAPI().GetFiltered(rDevices); err == nil {
		w.Header().Set("Content-Type", api.ContentType_JSON)
		enc := json.NewEncoder(w)
		if err = enc.Encode(lic); err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	} else {
		switch err {
		case webpublication.ErrNotFound:
			{
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			}
		default:
			{
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			}
		}
	}
}

// GetLicense gets an existing license by its id (passed as a section of the REST URL).
// It generates a partial license from the purchase info,
// fetches the license from the lcp server and returns it to the caller.
// This API method is called from a link in the license status document.
//
func GetLicense(w http.ResponseWriter, r *http.Request, s IServer) {

	var purchase webpurchase.Purchase
	var err error

	vars := mux.Vars(r)
	var licenseID = vars["license_id"]
	// get the license id in the URL
	if purchase, err = s.PurchaseAPI().GetByLicenseID(licenseID); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	// get an existing license from the lcp server
	fullLicense, err := s.PurchaseAPI().GenerateOrGetLicense(purchase)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// return a json payload
	w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
	// the file name is license.lcpl
	w.Header().Set("Content-Disposition", "attachment; filename=\"license.lcpl\"")

	// returns the full license to the caller
	enc := json.NewEncoder(w)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(fullLicense)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// message to the console
	log.Println("Get license / id " + licenseID + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
}
