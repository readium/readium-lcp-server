/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package lutserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/store"

	"github.com/Machiel/slugify"
	"github.com/jinzhu/gorm"
)

// GetPurchases searches all purchases for a client
//
func GetPurchases(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	var err error

	pagination, err := ExtractPaginationFromRequest(req)
	if err != nil {
		// user id is not a number
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "Pagination error"}, http.StatusBadRequest)
		return
	}

	purchases, err := server.Store().Purchase().List(pagination.PerPage, pagination.Page)

	PrepareListHeaderResponse(len(purchases), "/api/v1/purchases", pagination, resp)
	resp.Header().Set(api.HdrContentType, api.ContentTypeJson)

	enc := json.NewEncoder(resp)
	err = enc.Encode(purchases)
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// GetUserPurchases searches all purchases for a client
//
func GetUserPurchases(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	var err error
	var userId int64
	vars := mux.Vars(req)

	if userId, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// user id is not a number
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
		return
	}

	pagination, err := ExtractPaginationFromRequest(req)
	if err != nil {
		// user id is not a number
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "Pagination error"}, http.StatusBadRequest)
		return
	}
	purchases, err := server.Store().Purchase().ListByUser(userId, pagination.PerPage, pagination.Page)
	if err != nil {
		// user id is not a number
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	PrepareListHeaderResponse(len(purchases), "/api/v1/users/"+vars["user_id"]+"/purchases", pagination, resp)
	resp.Header().Set(api.HdrContentType, api.ContentTypeJson)

	enc := json.NewEncoder(resp)
	err = enc.Encode(purchases)
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// CreatePurchase creates a purchase in the database
//
func CreatePurchase(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	purchase, err := api.ReadPurchasePayload(req)
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "incorrect JSON Purchase " + err.Error()}, http.StatusBadRequest)
		return
	}

	// purchase ok
	if err = server.Store().Purchase().Add(purchase); err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	// publication added to db
	resp.WriteHeader(http.StatusCreated)

	if purchase.Type == store.LOAN {
		log.Println("user " + strconv.Itoa(int(purchase.User.ID)) + " lent publication " + strconv.Itoa(int(purchase.Publication.ID)) + " until " + purchase.EndDate.Time.String())
	} else {
		log.Println("user " + strconv.Itoa(int(purchase.User.ID)) + " bought publication " + strconv.Itoa(int(purchase.Publication.ID)))
	}
}

// GetPurchasedLicense generates a new license from the corresponding purchase id (passed as a section of the REST URL).
// It fetches the license from the lcp server and returns it to the caller.
// This API method is called from the client app (angular) when a license is requested after a purchase.
//
func GetPurchasedLicense(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	vars := mux.Vars(req)
	var id int64
	var err error

	if id, err = strconv.ParseInt(vars["id"], 10, 64); err != nil {
		// id is not an integer (int64)
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
		return
	}

	purchase, err := server.Store().Purchase().Get(id)
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusNotFound)
		return
	}
	// FIXME: calling the lsd server at this point is too heavy: the max end date should be in the db.
	// FIXME: call lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	fullLicense, err := generateOrGetLicense(purchase, server)
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}

	attachmentName := slugify.Slugify(purchase.Publication.Title)
	resp.Header().Set(api.HdrContentType, api.ContentTypeLcpJson)
	resp.Header().Set(api.HdrContentDisposition, "attachment; filename=\""+attachmentName+".lcpl\"")

	enc := json.NewEncoder(resp)
	// does not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(fullLicense)

	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
	// message to the console
	log.Println("Return license / id " + vars["id"] + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))

}

// GetPurchase gets a purchase by its id in the database
//
func GetPurchase(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	vars := mux.Vars(req)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
		return
	}

	purchase, err := server.Store().Purchase().Get(int64(id))
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	// FIXME: calling the lsd server at this point is too heavy: the max end date should be in the db.
	// FIXME: call lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	resp.Header().Set(api.HdrContentType, api.ContentTypeJson)
	// json encode the purchase info into the output stream
	enc := json.NewEncoder(resp)
	if err = enc.Encode(purchase); err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// GetPurchaseByLicenseID gets a purchase by a license id in the database
//
func GetPurchaseByLicenseID(resp http.ResponseWriter, req *http.Request, server api.IServer) {
	vars := mux.Vars(req)
	var err error
	purchase, err := server.Store().Purchase().GetByLicenseID(vars["licenseID"])
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	// purchase found
	resp.Header().Set(api.HdrContentType, api.ContentTypeJson)
	enc := json.NewEncoder(resp)
	if err = enc.Encode(purchase); err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// UpdatePurchase updates a purchase in the database
// Only updates the license id (uuid), start and end date, status
//
func UpdatePurchase(resp http.ResponseWriter, req *http.Request, server api.IServer) {

	vars := mux.Vars(req)
	var id int
	var err error

	// check that the purchase id is an integer
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: "The purchase id must be an integer"}, http.StatusBadRequest)
		return
	}
	newPurchase, err := api.ReadPurchasePayload(req)
	// parse the update info
	if err != nil {
		api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	// console
	log.Printf("Update purchase %v, license id %v, start %v, end %v, status %v", newPurchase.ID, newPurchase.LicenseUUID.String, newPurchase.StartDate, newPurchase.EndDate, newPurchase.Status)

	// update the purchase, license id, start and end dates, status
	if err := server.Store().Purchase().Update(&store.Purchase{
		ID:          int64(id),
		LicenseUUID: newPurchase.LicenseUUID,
		StartDate:   newPurchase.StartDate,
		EndDate:     newPurchase.EndDate,
		Status:      newPurchase.Status}); err != nil {

		switch err {
		case gorm.ErrRecordNotFound:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			api.Error(resp, req, server.DefaultSrvLang(), api.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}

	resp.WriteHeader(http.StatusOK)
}
