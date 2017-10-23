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

package index

import (
	"database/sql"
	"errors"
)

var NotFound = errors.New("Content not found")

type Index interface {
	Get(id string) (Content, error)
	Add(c Content) error
	Update(c Content) error
	List() func() (Content, error)
}

type Content struct {
	Id            string `json:"id"`
	EncryptionKey []byte `json:"-"`
	Location      string `json:"location"`
	Length        int64  `json:"length"` //not exported in license spec?
	Sha256        string `json:"sha256"` //not exported in license spec?
}

type dbIndex struct {
	db   *sql.DB
	get  *sql.Stmt
	add  *sql.Stmt
	list *sql.Stmt
}

func (i dbIndex) Get(id string) (Content, error) {
	records, err := i.get.Query(id)
	defer records.Close()
	if records.Next() {
		var c Content
		err = records.Scan(&c.Id, &c.EncryptionKey, &c.Location, &c.Length, &c.Sha256)
		return c, err
	}

	return Content{}, NotFound
}

func (i dbIndex) Add(c Content) error {
	add, err := i.db.Prepare("INSERT INTO content (id,encryption_key,location,length,sha256) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(c.Id, c.EncryptionKey, c.Location, c.Length, c.Sha256)
	return err
}

func (i dbIndex) Update(c Content) error {
	add, err := i.db.Prepare("UPDATE content SET encryption_key=? , location=?, length=?,sha256=? WHERE id=?")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(c.EncryptionKey, c.Location, c.Length, c.Sha256, c.Id)
	return err
}

func (i dbIndex) List() func() (Content, error) {
	rows, err := i.list.Query()
	if err != nil {
		return func() (Content, error) { return Content{}, err }
	}
	return func() (Content, error) {
		var c Content
		var err error
		if rows.Next() {
			err = rows.Scan(&c.Id, &c.EncryptionKey, &c.Location, &c.Length, &c.Sha256)
		} else {
			rows.Close()
			err = NotFound
		}
		return c, err
	}
}

func Open(db *sql.DB) (i Index, err error) {
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS content (" +
		"id varchar(255) PRIMARY KEY," +
		"encryption_key varchar(64) NOT NULL," +
		"location text NOT NULL," +
		"`length` bigint," +
		"sha256 varchar(64))")
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT id,encryption_key,location,length,sha256 FROM content WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}
	list, err := db.Prepare("SELECT id,encryption_key,location,length,sha256 FROM content")
	if err != nil {
		return
	}
	i = dbIndex{db, get, nil, list}
	return
}
