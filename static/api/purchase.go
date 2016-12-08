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
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/static/webpurchase"
	"github.com/readium/readium-lcp-server/static/webuser"
)

//DecodeJSONPurchase transforms a json string to a User struct
func DecodeJSONPurchase(r *http.Request) (webpurchase.Purchase, error) {
	var dec *json.Decoder
	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_JSON {
		dec = json.NewDecoder(r.Body)
	}
	purchase := webpurchase.Purchase{}
	err := dec.Decode(&purchase)
	return purchase, err
}

//GetPurchasesForUser searches all purchases for a client
func GetPurchasesForUser(w http.ResponseWriter, r *http.Request, s IServer) {
	//TODO
	problem.Error(w, r, problem.Problem{Detail: "Not implemented"}, http.StatusNotImplemented)
}

//CreatePurchase creates a purchase in the database
func CreatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	if err := webpurchase.DecodeJSONPurchase(r, &purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: "Decode JSON error: " + err.Error()}, http.StatusBadRequest)
		return
	}
	//check user
	vars := mux.Vars(r)
	var id int64
	var err error
	if id, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	} else {
		if id != purchase.User.UserID {
			problem.Error(w, r, problem.Problem{Detail: "User ID must correpond with userID in purchase"}, http.StatusBadRequest)
		}
	}

	//purchase in PUT  data  ok
	if err := s.PurchaseAPI().Add(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// user added to db
	w.WriteHeader(http.StatusCreated)
}

//GetPurchase gets a purchase by its ID in the database
func GetPurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["purchase_id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().Get(int64(id)); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err == nil {
		// send json of correctly encoded user info
		w.Header().Set("Content-Type", api.ContentType_JSON)
		w.WriteHeader(http.StatusOK)
		return
	}
	problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
}

//GetPurchaseByLicenseID gets a purchase by a LicenseID in the database
func GetPurchaseByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var err error

	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["licenseID"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err == nil {
		// send json of correctly encoded user info
		w.Header().Set("Content-Type", api.ContentType_JSON)
		w.WriteHeader(http.StatusOK)
		return
	}
	problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
}

//RenewLicenseByLicenseID searches a purchase by a LicenseID in the database, and
// contacts the tcp server in order to renew the license
func RenewLicenseByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var err error

	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["licenseID"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found,  get a renewed license from lcpserver
	if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_ = json.NewEncoder(pw).Encode(purchase.PartialLicense)
			pw.Close() // signal end writing partial license (POST)
		}()
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/licenses/"+vars["licenseID"], pr)
		Auth := config.Config.LcpUpdateAuth
		if Auth.Username != "" {
			req.SetBasicAuth(Auth.Username, Auth.Password)
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					// got new  license, return license
					w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					w.Write(data)
					return
				}
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + pb.Title}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}

}

//UpdatePurchase updates a purchase in the database
func UpdatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	//ID is a number, check user (json)
	if err := webpurchase.DecodeJSONPurchase(r, &purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	// user ok, id is a number, search purchase to update
	if _, err := s.PurchaseAPI().Get(int64(id)); err == nil {
		// purchase found
		if err := s.PurchaseAPI().Update(webpurchase.Purchase{PurchaseID: int64(id), User: webuser.User{UserID: purchase.User.UserID}, Resource: purchase.Resource, TransactionDate: purchase.TransactionDate, PartialLicense: purchase.PartialLicense}); err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
	} else {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
}
