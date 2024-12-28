// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/dbutils"
)

var ErrNotFound = errors.New("License not found")

type Store interface {
	ListAll(page int, pageNum int) func() (LicenseReport, error)
	ListByContentID(ContentID string, page int, pageNum int) func() (LicenseReport, error)
	UpdateRights(l License) error
	Update(l License) error
	UpdateLsdStatus(id string, status int32) error
	Add(l License) error
	Get(id string) (License, error)
	TouchByContentID(ContentID string) error
}

type sqlStore struct {
	db                *sql.DB
	dbGetByID         *sql.Stmt
	dbList            *sql.Stmt
	dbListByContentID *sql.Stmt
}

// ListAll lists all licenses in ante-chronological order
// pageNum starts at 0
func (s *sqlStore) ListAll(pageSize int, pageNum int) func() (LicenseReport, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.LcpServer.Database)
	if driver == "mssql" {
		rows, err = s.dbList.Query(pageNum*pageSize, pageSize)
	} else {
		rows, err = s.dbList.Query(pageSize, pageNum*pageSize)
	}
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		var err error
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if rows.Next() {
			err = rows.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentID)
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return l, err
	}
}

// ListByContentID lists licenses for a given ContentID
// pageNum starting at 0
func (s *sqlStore) ListByContentID(contentID string, pageSize int, pageNum int) func() (LicenseReport, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.LcpServer.Database)
	if driver == "mssql" {
		rows, err = s.dbListByContentID.Query(contentID, pageNum*pageSize, pageSize)
	} else {
		rows, err = s.dbListByContentID.Query(contentID, pageSize, pageNum*pageSize)
	}
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		var err error
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if rows.Next() {
			err = rows.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentID)
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return l, err
	}
}

// UpdateRights
func (s *sqlStore) UpdateRights(l License) error {

	result, err := s.db.Exec(dbutils.GetParamQuery(config.Config.LcpServer.Database, "UPDATE license SET rights_print=?, rights_copy=?, rights_start=?, rights_end=?, updated=? WHERE id=?"),
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End, time.Now().UTC().Truncate(time.Second), l.ID)

	if err == nil {
		if r, _ := result.RowsAffected(); r == 0 {
			return ErrNotFound
		}
	}
	return err
}

// Add creates a new record in the license table
func (s *sqlStore) Add(l License) error {

	_, err := s.db.Exec(dbutils.GetParamQuery(config.Config.LcpServer.Database, `INSERT INTO license (id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?,  ?, ?)`),
		l.ID, l.User.ID, l.Provider, l.Issued, nil,
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End,
		l.ContentID)
	return err
}

// Update updates a record in the license table
func (s *sqlStore) Update(l License) error {

	_, err := s.db.Exec(dbutils.GetParamQuery(config.Config.LcpServer.Database, `UPDATE license SET user_id=?,provider=?,updated=?,
				rights_print=?,	rights_copy=?,	rights_start=?,	rights_end=?, content_fk =?
				WHERE id=?`),
		l.User.ID, l.Provider,
		time.Now().UTC().Truncate(time.Second),
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End,
		l.ContentID,
		l.ID)

	return err
}

// UpdateLsdStatus
func (s *sqlStore) UpdateLsdStatus(id string, status int32) error {

	_, err := s.db.Exec(dbutils.GetParamQuery(config.Config.LcpServer.Database, `UPDATE license SET lsd_status =? WHERE id=?`),
		status, id)

	return err
}

// Get a license from the db
func (s *sqlStore) Get(id string) (License, error) {

	row := s.dbGetByID.QueryRow(id)
	var l License
	l.Rights = new(UserRights)
	err := row.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
		&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentID)
	if err == sql.ErrNoRows {
		err = ErrNotFound
	}
	return l, err
}

// TouchByContentID updates the updated field of all licenses for a given contentID
func (s *sqlStore) TouchByContentID(contentID string) error {

	_, err := s.db.Exec(dbutils.GetParamQuery(config.Config.LcpServer.Database, `UPDATE license SET updated=? WHERE content_fk=?`),
		time.Now().UTC().Truncate(time.Second), contentID)
	if err != nil {
		log.Println("Error touching licenses for contentID", contentID)
	}
	return err
}

// Open
func Open(db *sql.DB) (store Store, err error) {

	driver, _ := config.GetDatabase(config.Config.LcpServer.Database)

	// if sqlite, create the license table if it does not exist
	if driver == "sqlite3" {
		_, err := db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating sqlite license table")
			return nil, err
		}
	}

	var dbList *sql.Stmt
	if driver == "mssql" {
		dbList, err = db.Prepare(dbutils.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated, rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license ORDER BY issued desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`))
	} else {
		dbList, err = db.Prepare(dbutils.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated, rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license ORDER BY issued desc LIMIT ? OFFSET ?`))
	}
	if err != nil {
		log.Println("Error preparing dbList")
		return
	}

	var dbListByContentID *sql.Stmt
	if driver == "mssql" {
		dbListByContentID, err = db.Prepare(dbutils.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated, 
		rights_print, rights_copy, rights_start, rights_end, content_fk
		FROM license WHERE content_fk = ? ORDER BY issued desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`))
	} else {
		dbListByContentID, err = db.Prepare(dbutils.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated, 
		rights_print, rights_copy, rights_start, rights_end, content_fk
		FROM license WHERE content_fk = ?  ORDER BY issued desc LIMIT ? OFFSET ?`))

	}
	if err != nil {
		log.Println("Error preparing dbListByContentID")
		return
	}

	var dbGetByID *sql.Stmt
	dbGetByID, err = db.Prepare(dbutils.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated, rights_print, rights_copy,
	rights_start, rights_end, content_fk 
	FROM license WHERE id = ?`))
	if err != nil {
		log.Println("Error preparing dbGetByID")
		return
	}

	store = &sqlStore{db, dbGetByID, dbList, dbListByContentID}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS license (" +
	"id varchar(255) PRIMARY KEY," +
	"user_id varchar(255) NOT NULL," +
	"provider varchar(255) NOT NULL," +
	"issued datetime NOT NULL," +
	"updated datetime DEFAULT NULL," +
	"rights_print int(11) DEFAULT NULL," +
	"rights_copy int(11) DEFAULT NULL," +
	"rights_start datetime DEFAULT NULL," +
	"rights_end datetime DEFAULT NULL," +
	"content_fk varchar(255) NOT NULL," +
	"lsd_status integer default 0," +
	"FOREIGN KEY(content_fk) REFERENCES content(id))"
