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
	"log"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"

	"github.com/jinzhu/gorm"
)

// GetPurchases searches all purchases for a client
//
func GetPurchases(server http.IServer, resp http.ResponseWriter, req *http.Request) (model.PurchaseCollection, error) {
	var err error

	pagination, err := ExtractPaginationFromRequest(req)
	if err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: "Pagination error", Status: http.StatusBadRequest}

	}

	purchases, err := server.Store().Purchase().List(pagination.PerPage, pagination.Page)

	PrepareListHeaderResponse(len(purchases), "/api/v1/purchases", pagination, resp)

	return purchases, nil
}

// GetUserPurchases searches all purchases for a client
//
func GetUserPurchases(server http.IServer, resp http.ResponseWriter, req *http.Request) (model.PurchaseCollection, error) {
	var err error
	var userId int64
	vars := mux.Vars(req)

	if userId, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest}

	}

	pagination, err := ExtractPaginationFromRequest(req)
	if err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: "Pagination error", Status: http.StatusBadRequest}
	}
	purchases, err := server.Store().Purchase().ListByUser(userId, pagination.PerPage, pagination.Page)
	if err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	PrepareListHeaderResponse(len(purchases), "/api/v1/users/"+vars["user_id"]+"/purchases", pagination, resp)

	return purchases, nil
}

// CreatePurchase creates a purchase in the database
//
func CreatePurchase(server http.IServer, payload *model.Purchase) (*string, error) {

	if err := server.Store().Purchase().Add(payload); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	if payload.Type == model.LOAN {
		log.Println("user " + strconv.Itoa(int(payload.User.ID)) + " lent publication " + strconv.Itoa(int(payload.Publication.ID)) + " until " + payload.EndDate.Time.String())
	} else {
		log.Println("user " + strconv.Itoa(int(payload.User.ID)) + " bought publication " + strconv.Itoa(int(payload.Publication.ID)))
	}
	return nil, http.Problem{Status: http.StatusCreated}
}

// GetPurchasedLicense generates a new license from the corresponding purchase id (passed as a section of the REST URL).
// It fetches the license from the lcp server and returns it to the caller.
// This API method is called from the client app (angular) when a license is requested after a purchase.
//
func GetPurchasedLicense(server http.IServer, resp http.ResponseWriter, req *http.Request) (*model.License, error) {
	vars := mux.Vars(req)
	var id int64
	var err error

	if id, err = strconv.ParseInt(vars["id"], 10, 64); err != nil {
		// id is not an integer (int64)
		return nil, http.Problem{Detail: "Purchase ID must be an integer", Status: http.StatusBadRequest}

	}

	purchase, err := server.Store().Purchase().Get(id)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}

	}
	// FIXME: calling the lsd server at this point is too heavy: the max end date should be in the db.
	// FIXME: call lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	fullLicense, err := generateOrGetLicense(purchase, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	attachmentName := http.Slugify(purchase.Publication.Title)
	resp.Header().Set(http.HdrContentType, http.ContentTypeLcpJson)
	resp.Header().Set(http.HdrContentDisposition, "attachment; filename=\""+attachmentName+".lcpl\"")

	// message to the console
	log.Println("Return license / id " + vars["id"] + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
	return fullLicense, nil
}

// GetPurchase gets a purchase by its id in the database
//
func GetPurchase(server http.IServer, resp http.ResponseWriter, req *http.Request) (*model.Purchase, error) {
	vars := mux.Vars(req)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		return nil, http.Problem{Detail: "Purchase ID must be an integer", Status: http.StatusBadRequest}

	}

	purchase, err := server.Store().Purchase().Get(int64(id))
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}

	}
	// FIXME: calling the lsd server at this point is too heavy: the max end date should be in the db.
	// FIXME: call lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	return purchase, nil
}

// GetPurchaseByLicenseID gets a purchase by a license id in the database
//
func GetPurchaseByLicenseID(server http.IServer, resp http.ResponseWriter, req *http.Request) (*model.Purchase, error) {
	vars := mux.Vars(req)
	var err error
	purchase, err := server.Store().Purchase().GetByLicenseID(vars["licenseID"])
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}

	}
	return purchase, nil
}

// UpdatePurchase updates a purchase in the database
// Only updates the license id (uuid), start and end date, status
//
func UpdatePurchase(server http.IServer, payload *model.Purchase, param ParamId) (*string, error) {

	var id int
	var err error

	// check that the purchase id is an integer
	if id, err = strconv.Atoi(param.Id); err != nil {
		return nil, http.Problem{Detail: "The purchase id must be an integer", Status: http.StatusBadRequest}

	}

	// console
	log.Printf("Update purchase %v, license id %v, start %v, end %v, status %v", payload.ID, payload.LicenseUUID.String, payload.StartDate, payload.EndDate, payload.Status)

	// update the purchase, license id, start and end dates, status
	if err := server.Store().Purchase().Update(&model.Purchase{
		ID:          int64(id),
		LicenseUUID: payload.LicenseUUID,
		StartDate:   payload.StartDate,
		EndDate:     payload.EndDate,
		Status:      payload.Status}); err != nil {

		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}

	}
	return nil, http.Problem{Status: http.StatusOK}
}
