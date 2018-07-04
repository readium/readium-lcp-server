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

package model_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
)

func TestUserStore_Add(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Unix())))
	salt := []byte(strconv.Itoa(r.Int()))
	magic := []byte("$" + "$")
	// create 100 different users. Password is the same as email
	for i := 0; i < 100; i++ {
		user := &model.User{}
		user.Email = faker.Internet().Email()
		user.Name = faker.Name().Name()
		user.Hint = faker.Name().FirstName()
		user.Password = string(http.MD5Crypt([]byte(user.Email), salt, magic))
		err := stor.User().Add(user)
		if err != nil {
			t.Fatalf("Error creating user : %v", err)
		}
	}
	// create a user that we can find by email
	user := &model.User{}
	user.Email = "antone@bashirian.name"
	user.Name = faker.Name().Name()
	user.Hint = faker.Name().FirstName()
	user.Password = string(http.MD5Crypt([]byte(user.Email), salt, magic))
	err := stor.User().Add(user)
	if err != nil {
		t.Fatalf("Error creating user : %v", err)
	}
}

func TestUserStore_ListUsers(t *testing.T) {
	users, err := stor.User().List(30, 0)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	for idx, user := range users {
		t.Logf("%d. %#v", idx, user)
	}

	users, err = stor.User().List(50, 1)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	for idx, user := range users {
		t.Logf("%d. %#v", idx, user)
	}
}

func TestUserStore_Get(t *testing.T) {
	user, err := stor.User().Get(1)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by id : %v", err)
		} else {
			t.Skipf("You forgot to create user with id = 1 ?")
		}
	}
	t.Logf("Found user by id : %#v", user)
}

func TestUserStore_GetByEmail(t *testing.T) {
	user, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by email : %v", err)
		} else {
			t.Skipf("You forgot to create user with email `antone@bashirian.name` ?")
		}
	}
	t.Logf("Found user by email : %#v", user)
}

func TestUserStore_Update(t *testing.T) {
	user, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving user by email : %v", err)
		} else {
			t.Skipf("You forgot to create user with email `antone@bashirian.name` ?")
		}
	}
	user.Name = "Updated user"
	user.Hint = "Updated hint"
	err = stor.User().Update(user)
	if err != nil {
		t.Fatalf("Error updating user by email : %v", err)
	}
	// reading it once again
	updatedUser, err := stor.User().GetByEmail("antone@bashirian.name")
	if err != nil {
		t.Fatalf("Error retrieving user by email : %v", err)
	}
	if updatedUser.Name != "Updated user" || updatedUser.Hint != "Updated hint" || updatedUser.Password != user.Password {
		t.Fatalf("Error : updated user is incorrect!\ngot %#v\nwant %#v", updatedUser, user)
	}
	t.Logf("Updated user : %#v", updatedUser)
}

func TestUserStore_DeleteUser(t *testing.T) {
	users, err := stor.User().List(1, 0)
	if err != nil {
		t.Fatalf("Error retrieving users : %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("No users found. Create some users.")
	}
	// deleting the first user
	err = stor.User().Delete(users[0].ID)
	if err != nil {
		t.Fatalf("Error deleting user : %v", err)
	}
	t.Logf("User deleted : %#v", users[0])
}
