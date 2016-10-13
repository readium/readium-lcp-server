// Copyright (c) 2016 Readium Founation
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
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/readium/readium-lcp-server/sign"
)

var NotFound = errors.New("Package not found")

type Store interface {
	List() func() (License, error)
	Add(l License) error
	Get(id string) (License, error)
}

type sqlStore struct {
	db *sql.DB
}

func (s *sqlStore) List() func() (License, error) {
	return func() (License, error) {
		return License{}, NotFound
	}
}

func (s *sqlStore) Add(l License) error {
	json, err := sign.Canon(l)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO licenses VALUES (?, ?)", l.Id, json)

	return err
}

func (s *sqlStore) Get(id string) (License, error) {
	var l License
	var buf []uint8

	row := s.db.QueryRow("SELECT data FROM licenses where id = ?", id)

	err := row.Scan(&buf)
	if err != nil {
		return l, err
	}

	b := bytes.NewBuffer(buf)

	dec := json.NewDecoder(b)
	err = dec.Decode(&l)
	if err != nil {
		return l, err
	}

	return l, nil
}

func NewSqlStore(db *sql.DB) (Store, error) {
	_, err := db.Exec(tableDef)
	if err != nil {
		return nil, err
	}

	return &sqlStore{db}, nil
}

const tableDef = `CREATE TABLE IF NOT EXISTS licenses (
	id varchar(255) PRIMARY KEY,
	data blob)`
