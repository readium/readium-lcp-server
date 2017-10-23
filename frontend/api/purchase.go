// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package staticapi

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"

	"github.com/Machiel/slugify"
)

// DecodeJSONPurchase transforms a json object into an golang object
//
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
//
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

	PrepareListHeaderResponse(len(purchases), "/api/v1/purchases", pagination, w)
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	err = enc.Encode(purchases)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// GetUserPurchases searches all purchases for a client
//
func GetUserPurchases(w http.ResponseWriter, r *http.Request, s IServer) {
	var err error
	var userId int64
	vars := mux.Vars(r)

	if userId, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// user id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
		return
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

	PrepareListHeaderResponse(len(purchases), "/api/v1/users/"+vars["user_id"]+"/purchases", pagination, w)
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	err = enc.Encode(purchases)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// CreatePurchase creates a purchase in the database
//
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

	if purchase.Type == webpurchase.LOAN {
		log.Println("user " + strconv.Itoa(int(purchase.User.ID)) + " lent publication " + strconv.Itoa(int(purchase.Publication.ID)) + " until " + purchase.EndDate.String())
	} else {
		log.Println("user " + strconv.Itoa(int(purchase.User.ID)) + " bought publication " + strconv.Itoa(int(purchase.Publication.ID)))
	}
}

// GetPurchasedLicense gets a license from the corresponding purchase id (passed as a section of the REST URL).
// It fetches the license from the lcp server and returns it to the caller.
// This API method is called from the client app (angular?js) when a license is requested after a purchase.
//
func GetPurchasedLicense(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	var id int64
	var err error

	if id, err = strconv.ParseInt(vars["id"], 10, 64); err != nil {
		// id is not an integer (int64)
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
		return
	}

	purchase, err := s.PurchaseAPI().Get(id)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		return
	}

	fullLicense, err := s.PurchaseAPI().GenerateLicense(purchase)
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
	// message to the console
	log.Println("Return license / id " + vars["id"] + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))

}

// GetPurchase gets a purchase by its id in the database
//
func GetPurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
		return
	}

	purchase, err := s.PurchaseAPI().Get(int64(id))
	if err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", api.ContentType_JSON)
	// json encode the purchase info into the output stream
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// GetPurchaseByLicenseID gets a purchase by a license id in the database
//
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
		return
	}
	// purchase found
	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		return
	}
}

// getLicenseInfo decodes a license in data (bytes, response.body)
// FIXME : seems unused
//
func getLicenseInfo(data []byte, lic *license.License) error {
	var dec *json.Decoder
	dec = json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&lic); err != nil {
		return err
	}
	return nil
}

// UpdatePurchase updates a purchase in the database
// Only updates the license id (uuid), start and end date, status
//
func UpdatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var newPurchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error

	// check that the purchase id is an integer
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		problem.Error(w, r, problem.Problem{Detail: "The purchase id must be an integer"}, http.StatusBadRequest)
		return
	}
	// parse the update info
	if newPurchase, err = DecodeJSONPurchase(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// update the purchase, license id, start and end dates, status.
	if err := s.PurchaseAPI().Update(webpurchase.Purchase{
		ID:          int64(id),
		LicenseUUID: newPurchase.LicenseUUID,
		StartDate:   newPurchase.StartDate,
		EndDate:     newPurchase.EndDate,
		Status:      newPurchase.Status}); err != nil {

		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)

	// console
	log.Printf("Update purchase %s, license id %v, start %v, end %v, status %v", vars["id"], newPurchase.LicenseUUID, newPurchase.StartDate, newPurchase.EndDate, newPurchase.Status)
}
