// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package webuser

import (
	"database/sql"
	"errors"
	"log"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/dbutils"
	uuid "github.com/satori/go.uuid"
)

// ErrNotFound error trown when user is not found
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

// User struct defines a user
type User struct {
	ID       int64  `json:"id"`
	UUID     string `json:"uuid"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Hint     string `json:"hint"`
}

type dbUser struct {
	db           *sql.DB
	dbGetUser    *sql.Stmt
	dbGetByEmail *sql.Stmt
	dbList       *sql.Stmt
}

// Get returns a user
func (user dbUser) Get(id int64) (User, error) {

	row := user.dbGetUser.QueryRow(id)
	var c User
	err := row.Scan(&c.ID, &c.UUID, &c.Name, &c.Email, &c.Password, &c.Hint)
	if err != nil {
		return User{}, ErrNotFound
	}
	return c, err
}

// GetByEmail returns a user
func (user dbUser) GetByEmail(email string) (User, error) {

	row := user.dbGetByEmail.QueryRow(email)
	var c User
	err := row.Scan(&c.ID, &c.UUID, &c.Name, &c.Email, &c.Password, &c.Hint)
	return c, err
}

// Add inserts a user
func (user dbUser) Add(newUser User) error {

	// Create uuid
	uid, err_u := uuid.NewV4()
	if err_u != nil {
		return err_u
	}
	newUser.UUID = uid.String()

	_, err := user.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database,
		"INSERT INTO \"user\" (uuid, name, email, password, hint) VALUES (?, ?, ?, ?, ?)"),
		newUser.UUID, newUser.Name, newUser.Email, newUser.Password, newUser.Hint)
	return err
}

// Update updates a user
func (user dbUser) Update(changedUser User) error {

	_, err := user.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database,
		"UPDATE \"user\" SET name=? , email=?, password=?, hint=? WHERE id=?"),
		changedUser.Name, changedUser.Email, changedUser.Password, changedUser.Hint, changedUser.ID)
	return err
}

// DeleteUser deletes a user
func (user dbUser) DeleteUser(userID int64) error {

	// delete user purchases
	_, err := user.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database, "DELETE FROM purchase WHERE user_id=?"), userID)
	if err != nil {
		return err
	}

	// delete user
	_, err = user.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database, "DELETE FROM \"user\" WHERE id=?"), userID)
	return err
}

// ListUsers lists users
func (user dbUser) ListUsers(page int, pageNum int) func() (User, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	if driver == "mssql" {
		rows, err = user.dbList.Query(pageNum*page, page)
	} else {
		rows, err = user.dbList.Query(page, pageNum*page)
	}
	if err != nil {
		return func() (User, error) { return User{}, err }
	}

	return func() (User, error) {
		var u User
		var err error
		if rows.Next() {
			err = rows.Scan(&u.ID, &u.UUID, &u.Name, &u.Email, &u.Password, &u.Hint)
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return u, err
	}
}

// Open  returns a WebUser interface (db interaction)
func Open(db *sql.DB) (i WebUser, err error) {

	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	// if sqlite, create the content table in the frontend db if it does not exist
	if driver == "sqlite3" {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating user table")
			return
		}
	}

	var dbGetUser *sql.Stmt
	dbGetUser, err = db.Prepare(dbutils.GetParamQuery(config.Config.FrontendServer.Database,
		"SELECT id, uuid, name, email, password, hint FROM \"user\" WHERE id = ?"))
	if err != nil {
		return
	}

	var dbGetByEmail *sql.Stmt
	if driver == "mssql" {
		dbGetByEmail, err = db.Prepare("SELECT TOP 1 id, uuid, name, email, password, hint FROM \"user\" WHERE email = ?")
	} else {
		dbGetByEmail, err = db.Prepare(dbutils.GetParamQuery(config.Config.FrontendServer.Database,
			"SELECT id, uuid, name, email, password, hint FROM \"user\" WHERE email = ? LIMIT 1"))
	}
	if err != nil {
		return
	}

	var dbList *sql.Stmt
	if driver == "mssql" {
		dbList, err = db.Prepare("SELECT id, uuid, name, email, password, hint	FROM \"user\" ORDER BY email desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY")
	} else {
		dbList, err = db.Prepare(dbutils.GetParamQuery(config.Config.FrontendServer.Database,
			"SELECT id, uuid, name, email, password, hint	FROM \"user\" ORDER BY email desc LIMIT ? OFFSET ?"))
	}
	if err != nil {
		return
	}

	i = dbUser{db, dbGetUser, dbGetByEmail, dbList}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS \"user\" (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"name varchar(64) NOT NULL," +
	"email varchar(64) NOT NULL," +
	"password varchar(64) NOT NULL," +
	"hint varchar(64) NOT NULL)"
