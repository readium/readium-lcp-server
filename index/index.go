// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package index

import (
	"database/sql"
	"errors"
	"log"

	"github.com/readium/readium-lcp-server/config"
)

// ErrNotFound signals content not found
var ErrNotFound = errors.New("Content not found")

// Index is an interface
type Index interface {
	Get(id string) (Content, error)
	Add(c Content) error
	Update(c Content) error
	List() func() (Content, error)
}

// Content represents an encrypted resource
type Content struct {
	ID            string `json:"id"`
	EncryptionKey []byte `json:"-"`
	Location      string `json:"location"`
	Length        int64  `json:"length"` //not exported in license spec?
	Sha256        string `json:"sha256"` //not exported in license spec?
	Type          string `json:"type"`
}

type dbIndex struct {
	db        *sql.DB
	dbGetByID *sql.Stmt
	dbList    *sql.Stmt
}

// Get returns a record by id
func (i dbIndex) Get(id string) (Content, error) {
	row := i.dbGetByID.QueryRow(id)
	var c Content
	err := row.Scan(&c.ID, &c.EncryptionKey, &c.Location, &c.Length, &c.Sha256, &c.Type)
	if err != nil {
		err = ErrNotFound
	}
	return c, err
}

// Add inserts a record
func (i dbIndex) Add(c Content) error {
	_, err := i.db.Exec("INSERT INTO content (id,encryption_key,location,length,sha256,type) VALUES (?, ?, ?, ?, ?, ?)",
		c.ID, c.EncryptionKey, c.Location, c.Length, c.Sha256, c.Type)
	return err
}

// Update updates a record
func (i dbIndex) Update(c Content) error {
	_, err := i.db.Exec("UPDATE content SET encryption_key=? , location=?, length=?, sha256=?, type=? WHERE id=?",
		c.EncryptionKey, c.Location, c.Length, c.Sha256, c.Type, c.ID)
	return err
}

// List lists rows
func (i dbIndex) List() func() (Content, error) {
	rows, err := i.dbList.Query()
	if err != nil {
		return func() (Content, error) { return Content{}, err }
	}
	return func() (Content, error) {
		var c Content
		var err error
		if rows.Next() {
			err = rows.Scan(&c.ID, &c.EncryptionKey, &c.Location, &c.Length, &c.Sha256, &c.Type)
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return c, err
	}
}

// Open opens an SQL database and prepare db statements
func Open(db *sql.DB) (i Index, err error) {
	driver, _ := config.GetDatabase(config.Config.LcpServer.Database)

	// if sqlite, create the content table in the lcp db if it does not exist
	if driver == "sqlite3" {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating sqlite content table")
			return
		}
	}

	var dbGetByID *sql.Stmt
	dbGetByID, err = db.Prepare("SELECT id,encryption_key,location,length,sha256,type FROM content WHERE id = ?")
	if err != nil {
		return
	}
	dbList, err := db.Prepare("SELECT id,encryption_key,location,length,sha256,type FROM content")
	if err != nil {
		return
	}
	i = dbIndex{db, dbGetByID, dbList}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS content (" +
	"id varchar(255) PRIMARY KEY," +
	"encryption_key varchar(64) NOT NULL," +
	"location text NOT NULL," +
	"length bigint," +
	"sha256 varchar(64)," +
	"\"type\" varchar(255) NOT NULL default 'application/epub+zip')"
