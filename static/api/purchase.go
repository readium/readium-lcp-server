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
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/static/webpurchase"
	"github.com/readium/readium-lcp-server/static/webuser"
)

/*TODO searches purchases for a client
func GetPurchasesForUser(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
}
*/

//CreatePurchase creates a purchase in the database
func CreatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	if err := webpurchase.DecodeJSONPurchase(r, &purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	//purchase in PUT  data  ok
	if err := s.PurchaseAPI().Add(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// user added to db
	w.WriteHeader(http.StatusCreated)
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
		// purchase found!

		if err := s.PurchaseAPI().Update(webpurchase.Purchase{PurchaseID: int64(id), User: webuser.User{UserID: purchase.User.UserID}, Resource: purchase.Resource, TransactionDate: purchase.TransactionDate, PartialLicense: purchase.PartialLicense}); err != nil {
			//update failed!
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		//return
	} else {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
}
