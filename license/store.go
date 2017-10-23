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

package license

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
)

var NotFound = errors.New("License not found")

type Store interface {
	//List() func() (License, error)
	List(ContentId string, page int, pageNum int) func() (LicenseReport, error)
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

// notifyLsdServer informs LSD server of a new License
// and saves the result of the http request in the DB (using the *Store)
//
func notifyLsdServer(l License, s Store) {
	if config.Config.LsdServer.PublicBaseUrl != "" {
		var lsdClient = &http.Client{
			Timeout: time.Second * 10,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_ = json.NewEncoder(pw).Encode(l)
			pw.Close() // signal end writing
		}()
		req, err := http.NewRequest("PUT", config.Config.LsdServer.PublicBaseUrl+"/licenses", pr)

		// Set credentials on lsd request
		notifyAuth := config.Config.LsdNotifyAuth
		if notifyAuth.Username != "" {
			req.SetBasicAuth(notifyAuth.Username, notifyAuth.Password)
		}

		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

		response, err := lsdClient.Do(req)
		if err != nil {
			log.Println("Error Notify LsdServer of new License (" + l.Id + "):" + err.Error())
			_ = s.UpdateLsdStatus(l.Id, -1)
		} else {
			defer req.Body.Close()
			_ = s.UpdateLsdStatus(l.Id, int32(response.StatusCode))
			if response.StatusCode != 201 { //bad request or server error
				log.Println("Notify LsdServer of new License (" + l.Id + ") = " + strconv.Itoa(response.StatusCode))
			}
		}
	}
}

// ListAll lists all licenses in ante-chronological order
// pageNum starts at 0
//
func (s *sqlStore) ListAll(page int, pageNum int) func() (LicenseReport, error) {
	listLicenses, err := s.db.Query(`SELECT id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license
	ORDER BY issued desc LIMIT ? OFFSET ? `, page, pageNum*page)
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if listLicenses.Next() {
			err := listLicenses.Scan(&l.Id, &l.User.Id, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentId)

			if err != nil {
				return l, err
			}

		} else {
			listLicenses.Close()
			err = NotFound
		}
		return l, err
	}
}

// List() list licenses for a given ContentId
// pageNum starting at 0
//
func (s *sqlStore) List(ContentId string, page int, pageNum int) func() (LicenseReport, error) {
	listLicenses, err := s.db.Query(`SELECT id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end, content_fk
	FROM license
	WHERE content_fk=? LIMIT ? OFFSET ? `, ContentId, page, pageNum*page)
	if err != nil {
		return func() (LicenseReport, error) { return LicenseReport{}, err }
	}
	return func() (LicenseReport, error) {
		var l LicenseReport
		l.User = UserInfo{}
		l.Rights = new(UserRights)
		if listLicenses.Next() {

			err := listLicenses.Scan(&l.Id, &l.User.Id, &l.Provider, &l.Issued, &l.Updated,
				&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End, &l.ContentId)
			if err != nil {
				return l, err
			}
		} else {
			listLicenses.Close()
			err = NotFound
		}
		return l, err
	}
}

// UpdateRights
//
func (s *sqlStore) UpdateRights(l License) error {
	result, err := s.db.Exec("UPDATE license SET rights_print=?, rights_copy=?, rights_start=?, rights_end=?,updated=?  WHERE id=?",
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End, time.Now(), l.Id)

	if err == nil {
		if r, _ := result.RowsAffected(); r == 0 {
			return NotFound
		}
	}
	return err
}

// Add
//
func (s *sqlStore) Add(l License) error {
	_, err := s.db.Exec(`INSERT INTO license (id, user_id, provider, issued, updated,
	rights_print, rights_copy, rights_start, rights_end,
	user_key_hint, user_key_hash, user_key_algorithm, content_fk) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?,  ?, ?, ?, ?, ?)`,
		l.Id, l.User.Id, l.Provider, l.Issued, nil, l.Rights.Print, l.Rights.Copy, l.Rights.Start,
		l.Rights.End, l.Encryption.UserKey.Hint, l.Encryption.UserKey.Check,
		l.Encryption.UserKey.Key.Algorithm, l.ContentId)
	go notifyLsdServer(l, s)
	return err
}

// Update
//
func (s *sqlStore) Update(l License) error {
	_, err := s.db.Exec(`UPDATE license SET user_id=?,provider=?,issued=?,updated=?,
				rights_print=?,	rights_copy=?,	rights_start=?,	rights_end=?,
				user_key_hint=?, content_fk =?
				WHERE id=?`, // user_key_hash=?, user_key_algorithm=?,
		l.User.Id, l.Provider, l.Issued, time.Now(),
		l.Rights.Print, l.Rights.Copy, l.Rights.Start, l.Rights.End,
		l.Encryption.UserKey.Hint, l.ContentId,
		l.Id)

	return err
}

// UpdateLsdStatus
//
func (s *sqlStore) UpdateLsdStatus(id string, status int32) error {
	_, err := s.db.Exec(`UPDATE license SET lsd_status =?
				WHERE id=?`, // user_key_hash=?, user_key_algorithm=?,
		status,
		id)

	return err
}

// Get
//
func (s *sqlStore) Get(id string) (License, error) {

	var l License
	createForeigns(&l)

	row := s.db.QueryRow(`SELECT id, user_id, provider, issued, updated, rights_print, rights_copy,
	rights_start, rights_end, user_key_hint, user_key_hash, user_key_algorithm, content_fk FROM license
	where id = ?`, id)

	err := row.Scan(&l.Id, &l.User.Id, &l.Provider, &l.Issued, &l.Updated,
		&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End,
		&l.Encryption.UserKey.Hint, &l.Encryption.UserKey.Check, &l.Encryption.UserKey.Key.Algorithm,
		&l.ContentId)

	if err != nil {
		if err == sql.ErrNoRows {
			return l, NotFound
		} else {
			return l, err
		}
	}

	return l, nil
}

// NewSqlStore
//
func NewSqlStore(db *sql.DB) (Store, error) {
	_, err := db.Exec(tableDef)
	if err != nil {
		return nil, err
	}

	return &sqlStore{db}, nil
}

const tableDef = `CREATE TABLE IF NOT EXISTS license (
	id varchar(255) PRIMARY KEY,
	user_id varchar(255) NOT NULL,
	provider varchar(255) NOT NULL,
	issued datetime NOT NULL,
	updated datetime DEFAULT NULL,
	rights_print int(11) DEFAULT NULL,
	rights_copy int(11) DEFAULT NULL,
	rights_start datetime DEFAULT NULL,
	rights_end datetime DEFAULT NULL,
	user_key_hint text NOT NULL,
	user_key_hash varchar(64) NOT NULL,
	user_key_algorithm varchar(255) NOT NULL,
	content_fk varchar(255) NOT NULL,
	lsd_status integer default 0,
	FOREIGN KEY(content_fk) REFERENCES content(id))`
