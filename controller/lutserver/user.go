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

	"encoding/hex"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/model"
	"strings"
)

// GetUsers returns a paged list of users.
// If param.Filter is present, returns users list filtered by email
func GetUsers(server http.IServer, param ParamPagination) (*views.Renderer, error) {
	noOfUsers, err := server.Store().User().Count()
	if err != nil {
		return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	// Pagination
	page, perPage, err := http.ReadPagination(param.Page, param.PerPage, noOfUsers)
	if err != nil {
		return nil, http.Problem{Status: http.StatusBadRequest, Detail: err.Error()}
	}

	var users model.UsersCollection
	view := &views.Renderer{}
	if param.Filter != "" {
		view.AddKey("filter", param.Filter)
		noOfFilteredUsers, err := server.Store().User().FilterCount(param.Filter)
		if err != nil {
			return nil, http.Problem{Status: http.StatusInternalServerError, Detail: err.Error()}
		}
		view.AddKey("filterTotal", noOfFilteredUsers)
		users, err = server.Store().User().Filter(param.Filter, perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfFilteredUsers {
			view.AddKey("hasNextPage", true)
		}
		view.AddKey("noResults", noOfFilteredUsers == 0)
	} else {
		users, err = server.Store().User().List(perPage, page)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
		if (page+1)*perPage < noOfUsers {
			view.AddKey("hasNextPage", true)
		}
		view.AddKey("noResults", noOfUsers == 0)
	}
	view.AddKey("users", users)
	view.AddKey("pageTitle", "Users list")
	view.AddKey("total", noOfUsers)
	view.AddKey("currentPage", page+1)
	view.AddKey("perPage", perPage)
	view.Template("users/index.html.got")
	return view, nil
}

// GetUser returns an user by it's ID. If user ID is zero, we're displaying create form
func GetUser(server http.IServer, param ParamId) (*views.Renderer, error) {
	view := &views.Renderer{}
	var user *model.User
	if param.Id != "0" {
		id, err := strconv.Atoi(param.Id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest}
		}
		user, err = server.Store().User().Get(int64(id))
		if err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}

			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
		view.AddKey("pageTitle", "Edit user")
	} else {
		user = &model.User{}
		view.AddKey("pageTitle", "Create user")
	}

	view.AddKey("user", user)
	view.Template("users/form.html.got")
	return view, nil
}

// CreateOrUpdateUser creates/updates a user in the database
func CreateOrUpdateUser(server http.IServer, user *model.User) (*views.Renderer, error) {
	switch user.ID {
	case 0:
		if user.Password != "" {
			user.Password = hex.EncodeToString([]byte(user.Password))
		}
		err := server.Store().User().Add(user)
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
		}
	default:
		// searching for updated entity
		if existingUser, err := server.Store().User().Get(user.ID); err != nil {
			switch err {
			case gorm.ErrRecordNotFound:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
			default:
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		} else {
			updateEntity := &model.User{
				ID:       existingUser.ID,
				UUID:     existingUser.UUID,
				Name:     user.Name,
				Email:    user.Email,
				Password: user.Password,
				Hint:     user.Hint,
			}
			// only if a new password was provided, allow to change password / hint
			if user.Password == "" {
				updateEntity.Password = existingUser.Password
			} else {
				user.Password = hex.EncodeToString([]byte(user.Password))
			}
			if user.Hint == "" {
				updateEntity.Hint = existingUser.Hint
			}
			// performing update
			if err = server.Store().User().Update(updateEntity); err != nil {
				//update failed!
				return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
			}
		}
	}
	return nil, http.Problem{Detail: "/users", Status: http.StatusRedirect}
}

// Delete removes user from the database
func DeleteUser(server http.IServer, param ParamId) (*views.Renderer, error) {
	ids := strings.Split(param.Id, ",")
	var userIds []int64
	for _, id := range ids {
		uid, err := strconv.Atoi(id)
		if err != nil {
			// id is not a number
			return nil, http.Problem{Detail: "User ID must be an integer", Status: http.StatusBadRequest}
		}
		_, err = server.Store().User().Get(int64(uid))
		if err != nil {
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		}
		userIds = append(userIds, int64(uid))
	}
	if err := server.Store().User().BulkDelete(userIds); err != nil {
		return nil, http.Problem{Detail: err.Error(), Status: http.StatusBadRequest}
	}
	return nil, http.Problem{Status: http.StatusOK}
}
