// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

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
	if rDevices == "" {
		rDevices = "1"
	}

	if lic, err := s.LicenseAPI().GetFiltered(rDevices); err == nil {
		enc := json.NewEncoder(w)
		if err = enc.Encode(lic); err == nil {
			// send json of correctly encoded user info
			w.Header().Set("Content-Type", api.ContentType_JSON)
			w.WriteHeader(http.StatusOK)
			return
		}

		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
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

// GetLicense gets a license by its id (passed as a section of the REST URL).
// It fetches the license from the lcp server and returns it to the caller.
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
	// fetch the license from the lcp server
	fullLicense, err := s.PurchaseAPI().GenerateLicense(purchase)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
	// the file name is license.lcpl
	w.Header().Set("Content-Disposition", "attachment; filename=\"license.lcpl\"")

	// returns the license to the caller
	enc := json.NewEncoder(w)
	err = enc.Encode(fullLicense)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// message to the console
	log.Println("Get license / id " + licenseID + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
}
