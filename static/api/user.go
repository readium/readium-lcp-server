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
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/static/webpurchase"
	"github.com/readium/readium-lcp-server/static/webuser"
)

type IServer interface {
	UserAPI() webuser.WebUser
	PurchaseAPI() webpurchase.WebPurchase
}

//searches a client by his email
func GetUserByEmail(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	var email string
	email = vars["email"]
	if user, err := s.UserAPI().GetByEmail(email); err == nil {
		enc := json.NewEncoder(w)
		if err = enc.Encode(user); err == nil {
			// send json of correctly encoded user info
			w.Header().Set("Content-Type", api.ContentType_JSON)
			w.WriteHeader(http.StatusOK)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
	} else {
		switch err {
		case webuser.ErrNotFound:
			{
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
			}
		default:
			{
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			}
		}
	}
	return
}

//CreateUser creates a user in the database
func CreateUser(w http.ResponseWriter, r *http.Request, s IServer) {
	var user webuser.User

	if err := webuser.DecodeJsonUser(r, &user); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	//user ok
	if err := s.UserAPI().Add(user); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// user added to db
	w.WriteHeader(http.StatusCreated)
}

//UpdateUser updates an identified user (id) in the database
func UpdateUser(w http.ResponseWriter, r *http.Request, s IServer) {
	var user webuser.User
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	}
	//ID is a number, check user (json)
	if err := webuser.DecodeJsonUser(r, &user); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	// user ok, id is a number, search user to update
	if user, err := s.UserAPI().Get(int64(id)); err == nil {
		// client is found!

		if err := s.UserAPI().Update(webuser.User{UserID: id, Alias: user.Alias, Email: user.Email, Password: user.Password}); err != nil {
			//update failed!
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	if err := webuser.DecodeJsonUser(r, &user); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	//user ok
	if err := s.UserAPI().Add(user); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	// user added to db
	w.WriteHeader(http.StatusCreated)
}
