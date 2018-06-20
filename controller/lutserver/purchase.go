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
	"strconv"

	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"

	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/views"
	"strings"
	"time"
)

// GetUserPurchases searches all purchases for a client
//
func GetUserPurchases(server http.IServer, param ParamPaginationAndId) (*views.Renderer, error) {
	userId, err := strconv.ParseInt(param.Id, 10, 64)
	if err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest}
	}
	noOfPurchases, err := server.Store().Purchase().CountByUser(userId)
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfPurchases)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	purchases, err := server.Store().Purchase().ListByUser(userId, perPage, page)
	if err != nil {
		// user id is not a number
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
	}
	view := &views.Renderer{}
	view.AddKey("purchases", purchases)
	view.AddKey("pageTitle", "User purchases list")
	view.AddKey("total", noOfPurchases)
	view.AddKey("currentPage", page+1)
	view.AddKey("perPage", perPage)
	view.Template("purchases/index.html.got")
	return view, nil
}

// GetPurchasedLicense generates a new license from the corresponding purchase id (passed as a section of the REST URL).
// It fetches the license from the lcp server and returns it to the caller.
// This API method is called from the client app (angular) when a license is requested after a purchase.
//
func GetPurchasedLicense(server http.IServer, param ParamId) (*views.Renderer, error) {
	id, err := strconv.Atoi(param.Id)
	if err != nil {
		// id is not a number
		return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
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
	fullLicense, err := generateOrGetLicense(purchase, server)
	if err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}

	}

	attachmentName := http.Slugify(purchase.Publication.Title)
	server.LogInfo("Attachement name : %s\nFull license: %#v", attachmentName, fullLicense)
	//resp.Header().Set(http.HdrContentType, http.ContentTypeLcpJson)
	//resp.Header().Set(http.HdrContentDisposition, "attachment; filename=\""+attachmentName+".lcpl\"")

	// message to the console
	//log.Println("Return license / id " + vars["id"] + " / " + purchase.Publication.Title + " / purchase " + strconv.FormatInt(purchase.ID, 10))
	return nil, http.Problem{Detail: "/purchases", Status: http.StatusRedirect}
}

// GetPurchase gets a purchase by its id in the database
//
func GetPurchase(server http.IServer, param ParamId) (*views.Renderer, error) {
	view := &views.Renderer{}
	var purchase *model.Purchase
	if param.Id != "0" {
		id, err := strconv.Atoi(param.Id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "Publication ID must be an integer", Status: http.StatusBadRequest}
		}
		purchase, err = server.Store().Purchase().Get(int64(id))
		if err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}

			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
		view.AddKey("pageTitle", "Edit purchase")
	} else {
		existingPublications, err := server.Store().Publication().ListAll()
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		view.AddKey("existingPublications", existingPublications)

		existingUsers, err := server.Store().User().ListAll()
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		view.AddKey("existingUsers", existingUsers)
		// convention - if sID is zero, we're displaying create form
		purchase = &model.Purchase{Type: "Loan"}
		view.AddKey("pageTitle", "Create purchase")
	}
	view.AddKey("purchase", purchase)
	view.Template("purchases/form.html.got")
	return view, nil
}

// GetPurchaseByLicenseID gets a purchase by a license id in the database
//
func GetPurchaseByLicenseID(server http.IServer, param ParamPaginationAndId) (*views.Renderer, error) {
	var err error
	purchase, err := server.Store().Purchase().GetByLicenseID(param.Id)
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}

	}
	view := &views.Renderer{}
	view.AddKey("purchase", purchase)
	view.AddKey("pageTitle", "Edit purchase")
	return view, nil
}

// GetPurchases searches all purchases for a client
//
func GetPurchases(server http.IServer, param ParamPagination) (*views.Renderer, error) {
	var err error
	noOfPurchases, err := server.Store().Purchase().Count()
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfPurchases)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	var purchases model.PurchaseCollection
	view := &views.Renderer{}
	if param.Filter != "" {
		view.AddKey("filter", param.Filter)
		noOfFilteredPurchases, err := server.Store().Purchase().FilterCount(param.Filter)
		if err != nil {
			return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
		}
		view.AddKey("filterTotal", noOfFilteredPurchases)
		purchases, err = server.Store().Purchase().Filter(param.Filter, perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfFilteredPurchases {
			view.AddKey("hasNextPage", true)
		}
	} else {
		purchases, err = server.Store().Purchase().List(perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfPurchases {
			view.AddKey("hasNextPage", true)
		}
	}
	view.AddKey("purchases", purchases)
	view.AddKey("pageTitle", "Purchases list")
	view.AddKey("total", noOfPurchases)
	view.AddKey("currentPage", page+1)
	view.AddKey("perPage", perPage)
	view.Template("purchases/index.html.got")
	return view, nil
}

// CreatePurchase creates a purchase in the database
//
// TODO : shouldn't this notify LCP / LSD on create/update ?
func CreateOrUpdatePurchase(server http.IServer, payload *model.Purchase) (*views.Renderer, error) {
	switch payload.ID {
	case 0:
		if err := server.Store().Purchase().Add(payload); err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	default:
		if payload.StartDate.Valid {
			payload.StartDate.Time = payload.StartDate.Time.UTC().Truncate(time.Second)
		}

		if payload.EndDate.Valid {
			payload.EndDate.Time = payload.EndDate.Time.UTC().Truncate(time.Second)
		}

		// update the purchase, license id, start and end dates, status
		if err := server.Store().Purchase().Update(&model.Purchase{
			ID:        payload.ID,
			StartDate: payload.StartDate,
			EndDate:   payload.EndDate,
			Type:      payload.Type,
			Status:    payload.Status}); err != nil {

			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}

		}
	}

	return nil, http.Problem{Detail: "/purchases", Status: http.StatusRedirect}
}

// Delete removes purchase from the database
func DeletePurchase(server http.IServer, param ParamId) (*views.Renderer, error) {
	ids := strings.Split(param.Id, ",")
	var pendingDeleteIds []int64
	for _, id := range ids {
		uid, err := strconv.Atoi(id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "ID must be an integer", Status: http.StatusBadRequest}
		}
		_, err = server.Store().Purchase().Get(int64(uid))
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		}
		pendingDeleteIds = append(pendingDeleteIds, int64(uid))
	}
	if err := server.Store().Purchase().BulkDelete(pendingDeleteIds); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	return nil, http.Problem{Status: http.StatusOK}
}
