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
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
)

//GetUsers returns a list of users
func GetUsers(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	var page int64
	var perPage int64
	var err error
	if req.FormValue("page") != "" {
		page, err = strconv.ParseInt((req).FormValue("page"), 10, 32)
		if err != nil {
			server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		page = 1
	}
	if req.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((req).FormValue("per_page"), 10, 32)
		if err != nil {
			server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
			return
		}
	} else {
		perPage = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		server.Error(resp, req, http.Problem{Detail: "page must be positive integer", Status: http.StatusBadRequest})
		return
	}

	users, err := server.Store().User().List(int(perPage), int(page))
	if err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}

	if len(users) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		resp.Header().Set("Link", "</users/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}

	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		resp.Header().Set("Link", "</users/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}

	resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
	enc := json.NewEncoder(resp)
	err = enc.Encode(users)
	if err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		return
	}
}

//GetUser searches a client by his email
func GetUser(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	vars := mux.Vars(req)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		// id is not a number
		server.Error(resp, req, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest})
	}
	if user, err := server.Store().User().Get(int64(id)); err == nil {
		enc := json.NewEncoder(resp)
		if err = enc.Encode(user); err == nil {
			// send json of correctly encoded user info
			resp.Header().Set(http.HdrContentType, http.ContentTypeJson)
			resp.WriteHeader(http.StatusOK)
			return
		}
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusNotFound})
			}
		default:
			{
				server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			}
		}
	}
	return
}

//CreateUser creates a user in the database
func CreateUser(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	var user *model.User
	var err error
	if user, err = ReadUserPayload(req); err != nil {
		server.Error(resp, req, http.Problem{Detail: "incorrect JSON User " + err.Error(), Status: http.StatusBadRequest})
		return
	}
	//user ok
	if err = server.Store().User().Add(user); err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// user added to db
	resp.WriteHeader(http.StatusCreated)
}

//UpdateUser updates an identified user (id) in the database
func UpdateUser(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	vars := mux.Vars(req)
	var id int
	var err error
	var user *model.User
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		server.Error(resp, req, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest})
		return
	}
	//ID is a number, check user (json)
	if user, err = ReadUserPayload(req); err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// user ok, id is a number, search user to update
	if _, err = server.Store().User().Get(int64(id)); err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusNotFound})
		default:
			server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
		}
	} else {
		// client is found!
		if err = server.Store().User().Update(&model.User{ID: int64(id), Name: user.Name, Email: user.Email, Password: user.Password, Hint: user.Hint}); err != nil {
			//update failed!
			server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError})
			return
		}
		//database update ok
		resp.WriteHeader(http.StatusOK)
		//return
	}

}

//Delete creates a user in the database
func DeleteUser(resp http.ResponseWriter, req *http.Request, server http.IServer) {
	vars := mux.Vars(req)
	uid, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	if err = server.Store().User().Delete(uid); err != nil {
		server.Error(resp, req, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest})
		return
	}
	// user added to db
	resp.WriteHeader(http.StatusOK)
}

//ReadUserPayload transforms a json string to a User struct
func ReadUserPayload(req *http.Request) (*model.User, error) {
	var dec *json.Decoder
	if ctype := req.Header[http.HdrContentType]; len(ctype) > 0 && ctype[0] == http.ContentTypeJson {
		dec = json.NewDecoder(req.Body)
	}
	user := &model.User{}
	err := dec.Decode(user)
	return user, err
}
