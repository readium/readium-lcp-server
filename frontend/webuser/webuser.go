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

package webuser

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/readium/readium-lcp-server/config"
	"github.com/satori/go.uuid"
)

//ErrNotFound error trown when user is not found
var ErrNotFound = errors.New("User not found")

// WebUser interface for user db interaction
type WebUser interface {
	Get(id int64) (User, error)
	GetByEmail(email string) (User, error)
	Add(c User) error
	Update(c User) error
	DeleteUser(UserID int64) error
	ListUsers(page int, pageNum int) func() (User, error)
}

//User struct defines a user
type User struct {
	ID       int64  `json:"id"`
	UUID     string `json:"uuid"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Hint     string `json:"hint"`
}

type dbUser struct {
	db         *sql.DB
	getUser    *sql.Stmt
	getByEmail *sql.Stmt
}

func (user dbUser) Get(id int64) (User, error) {
	records, err := user.getUser.Query(id)
	defer records.Close()
	if records.Next() {
		var c User
		err = records.Scan(&c.ID, &c.UUID, &c.Name, &c.Email, &c.Password, &c.Hint)
		return c, err
	}

	return User{}, ErrNotFound
}

func (user dbUser) GetByEmail(email string) (User, error) {
	records, err := user.getByEmail.Query(email)
	defer records.Close()
	if records.Next() {
		var c User
		err = records.Scan(&c.ID, &c.UUID, &c.Name, &c.Email, &c.Password, &c.Hint)
		return c, err
	}

	return User{}, ErrNotFound
}

func (user dbUser) Add(newUser User) error {
	add, err := user.db.Prepare("INSERT INTO user (uuid, name, email, password, hint) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()

	// Create uuid
	uid, err_u := uuid.NewV4()
	if err_u != nil {
		return err_u
	}
	newUser.UUID = uid.String()

	_, err = add.Exec(newUser.UUID, newUser.Name, newUser.Email, newUser.Password, newUser.Hint)
	return err
}

func (user dbUser) Update(changedUser User) error {
	add, err := user.db.Prepare("UPDATE user SET name=? , email=?, password=?, hint=? WHERE id=?")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(changedUser.Name, changedUser.Email, changedUser.Password, changedUser.Hint, changedUser.ID)
	return err
}

func (user dbUser) DeleteUser(userID int64) error {
	// delete purchases from user
	delPurchases, err := user.db.Prepare(`DELETE FROM purchase WHERE user_id=?`)
	if err != nil {
		return err
	}
	defer delPurchases.Close()
	if _, err := delPurchases.Exec(userID); err != nil {
		return err
	}
	// and delete user
	query, err := user.db.Prepare("DELETE FROM user WHERE id=?")
	if err != nil {
		return err
	}
	defer query.Close()
	_, err = query.Exec(userID)
	return err
}

func (user dbUser) ListUsers(page int, pageNum int) func() (User, error) {
	listUsers, err := user.db.Query(`SELECT id, uuid, name, email, password, hint
	FROM user
	ORDER BY email desc LIMIT ? OFFSET ? `, page, pageNum*page)
	if err != nil {
		return func() (User, error) { return User{}, err }
	}
	return func() (User, error) {
		var u User
		if listUsers.Next() {
			err := listUsers.Scan(&u.ID, &u.UUID, &u.Name, &u.Email, &u.Password, &u.Hint)

			if err != nil {
				return u, err
			}

		} else {
			listUsers.Close()
			err = ErrNotFound
		}
		return u, err
	}
}

//Open  returns a WebUser interface (db interaction)
func Open(db *sql.DB) (i WebUser, err error) {
	// if sqlite, create the content table in the frontend db if it does not exist
	if strings.HasPrefix(config.Config.FrontendServer.Database, "sqlite") {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating user table")
			return
		}
	}
	get, err := db.Prepare("SELECT id, uuid, name, email, password, hint FROM user WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}
	getByEmail, err := db.Prepare("SELECT id, uuid, name, email, password, hint FROM user WHERE email = ? LIMIT 1")
	if err != nil {
		return
	}
	i = dbUser{db, get, getByEmail}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS user (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"name varchar(64) NOT NULL," +
	"email varchar(64) NOT NULL," +
	"password varchar(64) NOT NULL," +
	"hint varchar(64) NOT NULL)"
