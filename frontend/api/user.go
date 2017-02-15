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
	"github.com/readium/readium-lcp-server/frontend/webuser"
	"github.com/readium/readium-lcp-server/problem"
)

//GetUsers returns a list of users
func GetUsers(w http.ResponseWriter, r *http.Request, s IServer) {
	var page int64
	var perPage int64
	var err error
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}
	if r.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		perPage = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	users := make([]webuser.User, 0)
	//log.Println("ListAll(" + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.UserAPI().ListUsers(int(perPage), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		users = append(users, it)
	}
	if len(users) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</users/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</users/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	err = enc.Encode(users)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//GetUserByEmail searches a client by his email
func GetUser(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	}
	if user, err := s.UserAPI().Get(int64(id)); err == nil {
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

//DecodeJSONUser transforms a json string to a User struct
func DecodeJSONUser(r *http.Request) (webuser.User, error) {
	var dec *json.Decoder
	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_JSON {
		dec = json.NewDecoder(r.Body)
	}
	user := webuser.User{}
	err := dec.Decode(&user)
	return user, err
}

//CreateUser creates a user in the database
func CreateUser(w http.ResponseWriter, r *http.Request, s IServer) {
	var user webuser.User
	var err error
	if user, err = DecodeJSONUser(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: "incorrect JSON User " + err.Error()}, http.StatusBadRequest)
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
	vars := mux.Vars(r)
	var id int
	var err error
	var user webuser.User
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
		return
	}
	//ID is a number, check user (json)
	if user, err = DecodeJSONUser(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// user ok, id is a number, search user to update
	if _, err := s.UserAPI().Get(int64(id)); err != nil {
		switch err {
		case webuser.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	} else {
		// client is found!
		if err := s.UserAPI().Update(webuser.User{ID: int64(id), Name: user.Name, Email: user.Email, Password: user.Password}); err != nil {
			//update failed!
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			return
		}
		//database update ok
		w.WriteHeader(http.StatusOK)
		//return
	}

}

//DeleteUser creates a user in the database
func DeleteUser(w http.ResponseWriter, r *http.Request, s IServer) {
	vars := mux.Vars(r)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if err := s.UserAPI().DeleteUser(uid); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// user added to db
	w.WriteHeader(http.StatusOK)
}
