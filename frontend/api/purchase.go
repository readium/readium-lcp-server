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
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/frontend/webuser"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"

	"github.com/Machiel/slugify"
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

// GetPurchases searches all purchases for a client
func GetPurchases(w http.ResponseWriter, r *http.Request, s IServer) {
	var err error

	pagination, err := ExtractPaginationFromRequest(r)
	if err != nil {
		// user id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Pagination error"}, http.StatusBadRequest)
		return
	}

	purchases := make([]webpurchase.Purchase, 0)
	fn := s.PurchaseAPI().List(pagination.PerPage, pagination.Page)

	for it, err := fn(); err == nil; it, err = fn() {
		purchases = append(purchases, it)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(purchases)
	PrepareListHeaderResponse(len(purchases), "/api/v1/purchases", pagination, w)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//GetUserPurchases searches all purchases for a client
func GetUserPurchases(w http.ResponseWriter, r *http.Request, s IServer) {
	var err error
	var userId int64
	vars := mux.Vars(r)

	if userId, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// user id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	}

	pagination, err := ExtractPaginationFromRequest(r)
	if err != nil {
		// user id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Pagination error"}, http.StatusBadRequest)
		return
	}

	purchases := make([]webpurchase.Purchase, 0)
	fn := s.PurchaseAPI().ListByUser(userId, pagination.PerPage, pagination.Page)
	for it, err := fn(); err == nil; it, err = fn() {
		purchases = append(purchases, it)
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(purchases)
	PrepareListHeaderResponse(len(purchases), "/api/v1/users/"+vars["user_id"]+"/purchases", pagination, w)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//CreatePurchase creates a purchase in the database
func CreatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	var err error
	if purchase, err = DecodeJSONPurchase(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: "incorrect JSON Purchase " + err.Error()}, http.StatusBadRequest)
		return
	}

	// purchase ok
	if err = s.PurchaseAPI().Add(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// publication added to db
	w.WriteHeader(http.StatusCreated)
}

//GetPurchaseLicense contacts LCP server and asks a license for the purchase using the partial license and resourceID
func GetPurchaseLicense(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	var id int
	var err error

	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}

	purchase, err := s.PurchaseAPI().Get(int64(id))
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		return
	}

	fullLicense, err := s.PurchaseAPI().GetLicense(int64(id))
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	attachmentName := slugify.Slugify(purchase.Publication.Title)
	w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+attachmentName+".lcpl\"")

	enc := json.NewEncoder(w)
	err = enc.Encode(fullLicense)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

//GetPurchase gets a purchase by its ID in the database
func GetPurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}

	purchase, err := s.PurchaseAPI().Get(int64(id))
	if err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}

	// purchase found
	// purchase.PartialLicense = "*" //hide partialLicense?
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

// getLicenseInfo decoldes a license in data (bytes, response.body)
func getLicenseInfo(data []byte, lic *license.License) error {
	var dec *json.Decoder
	dec = json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&lic); err != nil {
		return err
	}
	return nil
}

//RenewLicenseByLicenseID searches a purchase by a LicenseID in the database, and
// contacts the tcp server in order to renew the license
func RenewLicenseByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	/*purchase := webpurchase.Purchase{}
	vars := mux.Vars(r)
	var err error
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["license_id"]); err != nil {
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
	}*/
	/*log.Println("POST " + config.Config.LcpServer.PublicBaseUrl + "/licenses/" + vars["license_id"])
		log.Println("BODY (partial license)")
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/licenses/"+vars["license_id"], strings.NewReader(purchase.PartialLicense))
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
					//forward headers
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
						}
					}
					// and license
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					w.Write(data)
					return
				}
			case 206:
				problem.Error(w, r, problem.Problem{Detail: "Partial content (invalid license) from LCP server"}, http.StatusNotFound)
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server : statuscode=" + strconv.Itoa(response.StatusCode)}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}*/

}

//RenewLicensePublicationByLicenseID searches a purchase by a LicenseID in the database, and
// contacts the tcp server in order to renew the license and return a publication with the license
// TODO !!!! (this actually returns the license= copy method RenewLicenseByLicenseID)
func RenewLicensePublicationByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	/*purchase := webpurchase.Purchase{}
	vars := mux.Vars(r)
	var err error
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["license_id"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}*/

	// purchase found,  get a renewed license from lcpserver
	/*if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		log.Println("POST " + config.Config.LcpServer.PublicBaseUrl + "/licenses/" + vars["license_id"] + "/publication")
		log.Println("BODY (partial license)")
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/licenses/"+vars["license_id"]+"/publication", strings.NewReader(purchase.PartialLicense))
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
					//forward headers
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
						}
					}
					// and publication/license
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					w.Write(data)
					return
				}
			case 206:
				problem.Error(w, r, problem.Problem{Detail: "Partial content (invalid license) from LCP server"}, http.StatusNotFound)
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server : statuscode=" + strconv.Itoa(response.StatusCode)}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}
	*/
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
	if purchase, err = DecodeJSONPurchase(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	// user ok, id is a number, search purchase to update
	if _, err := s.PurchaseAPI().Get(int64(id)); err == nil {
		// purchase found
		if err := s.PurchaseAPI().Update(webpurchase.Purchase{
			ID:              int64(id),
			User:            webuser.User{ID: purchase.User.ID},
			Publication:     webpublication.Publication{ID: purchase.Publication.ID},
			TransactionDate: purchase.TransactionDate}); err != nil {
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
