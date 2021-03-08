// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/database"
)

var ErrNotFound = errors.New("License not found")

type Store interface {
	//List() func() (License, error)
	List(ContentID string, page int, pageNum int) func() (LicenseReport, error)
	ListAll(page int, pageNum int) func() (LicenseReport, error)
	UpdateRights(l License) error
	Update(l License) error
	UpdateLsdStatus(id string, status int32) error
	Add(l License) error
	Get(id string) (License, error)
}

type sqlStore struct {
	db *sql.DB
}

// ListAll lists all licenses in ante-chronological order
// pageNum starts at 0
//
func (s *sqlStore) ListAll(page int, pageNum int) func() (LicenseReport, error) {
	listLicenses, err := s.db.Query(database.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license
    ORDER BY issued desc LIMIT ? OFFSET ? `), page, pageNum*page)
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if listLicenses.Next() {
			err := listLicenses.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentID)

			if err != nil {
				return l, err
			}

		} else {
			listLicenses.Close()
			err = ErrNotFound
		}
		return l, err
	}
}

// List lists licenses for a given ContentID
// pageNum starting at 0
//
func (s *sqlStore) List(contentID string, page int, pageNum int) func() (LicenseReport, error) {
	listLicenses, err := s.db.Query(database.GetParamQuery(config.Config.LcpServer.Database, `SELECT id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license
    WHERE content_fk=? LIMIT ? OFFSET ? `), contentID, page, pageNum*page)
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if listLicenses.Next() {

			err := listLicenses.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentID)
			if err != nil {
				return l, err
			}
		} else {
			listLicenses.Close()
			err = ErrNotFound
		}
		return l, err
	}
}

// UpdateRights
//
func (s *sqlStore) UpdateRights(l License) error {
	result, err := s.db.Exec(database.GetParamQuery(config.Config.LcpServer.Database, "UPDATE license SET rights_print=?, rights_copy=?, rights_start=?, rights_end=?,updated=?  WHERE id=?"),
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End, time.Now().UTC().Truncate(time.Second), l.ID)

	if err == nil {
		if r, _ := result.RowsAffected(); r == 0 {
			return ErrNotFound
		}
	}
	return err
}

// Add creates a new record in the license table
//
func (s *sqlStore) Add(l License) error {
	_, err := s.db.Exec(database.GetParamQuery(config.Config.LcpServer.Database, `INSERT INTO license (id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk) 
    VALUES (?, ?, ?, ?, ?, ?, ?, ?,  ?, ?)`),
		l.ID, l.User.ID, l.Provider, l.Issued, nil,
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End,
		l.ContentID)
	return err
}

// Update updates a record in the license table
//
func (s *sqlStore) Update(l License) error {
	_, err := s.db.Exec(database.GetParamQuery(config.Config.LcpServer.Database, `UPDATE license SET user_id=?,provider=?,updated=?,
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
//
func (s *sqlStore) UpdateLsdStatus(id string, status int32) error {
	_, err := s.db.Exec(database.GetParamQuery(config.Config.LcpServer.Database, `UPDATE license SET lsd_status =? WHERE id=?`),
		status,
		id)

	return err
}

// Get a license from the db
//
func (s *sqlStore) Get(id string) (License, error) {
	// create an empty license, add user rights
	var l License
	l.Rights = new(UserRights)

	row := s.db.QueryRow(`SELECT id, user_id, provider, issued, updated, rights_print, rights_copy,
	rights_start, rights_end, content_fk FROM license
	where id = ?`, id)

	err := row.Scan(&l.ID, &l.User.ID, &l.Provider, &l.Issued, &l.Updated,
		&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End,
		&l.ContentID)

	if err != nil {
		if err == sql.ErrNoRows {
			return l, ErrNotFound
		} else {
			return l, err
		}
	}

	return l, nil
}

// NewSqlStore
//
func NewSqlStore(db *sql.DB) (Store, error) {
	// if sqlite, create the license table if it does not exist
	if strings.HasPrefix(config.Config.LcpServer.Database, "sqlite") {
		_, err := db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating sqlite license table")
			return nil, err
		}
	}
	return &sqlStore{db}, nil
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
