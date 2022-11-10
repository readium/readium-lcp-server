// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/sign"
)

func TestCRUD(t *testing.T) {

	config.Config.LcpServer.Database = "sqlite3://:memory:"
	driver, cnxn := config.GetDatabase(config.Config.LcpServer.Database)
	db, err := sql.Open(driver, cnxn)
	if err != nil {
		t.Fatal(err)
	}

	st, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}

	fn := st.ListAll(10, 0)
	licenses := make([]LicenseReport, 0)
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) > 0 {
		t.Errorf("Failed getting an empty list")
	}

	l := License{}
	contentID := "1234-1234-1234-1234"
	Initialize(contentID, &l)

	l.User.ID = "me"
	l.Provider = "my.org"
	l.Rights = new(UserRights)
	rstart := time.Now().UTC().Truncate(time.Second)
	l.Rights.Start = &rstart
	rend := rstart.Add(time.Hour * 100)
	l.Rights.End = &rend
	rprint := int32(100)
	l.Rights.Print = &rprint
	rcopy := int32(1000)
	l.Rights.Copy = &rcopy

	err = st.Add(l)
	if err != nil {
		t.Fatal(err)
	}

	l2, err := st.Get(l.ID)
	if err != nil {
		t.Fatal(err)
	}

	js1, err := sign.Canon(l)
	js2, err2 := sign.Canon(l2)
	if err != nil || err2 != nil || !bytes.Equal(js1, js2) {
		t.Error("Difference between Add and Get")
	}

	// initializes another license with the same data
	Initialize(contentID, &l)
	err = st.Add(l)
	if err != nil {
		t.Fatal(err)
	}
	// and another with a different content id
	contentID2 := "5678-5678-5678-5678"
	Initialize(contentID2, &l)
	err = st.Add(l)
	if err != nil {
		t.Fatal(err)
	}

	// list all
	fn = st.ListAll(10, 0)
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) != 3 {
		t.Errorf("Failed getting three licenses; got %d licenses instead", len(licenses))
	}

	// list by content id
	licenses = make([]LicenseReport, 0)
	fn = st.ListByContentID(contentID, 10, 0)
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) != 2 {
		t.Errorf("Failed getting two licenses by contentID")
	}

	// update rights
	rstart = time.Now().UTC().Truncate(time.Second)
	l.Rights.Start = &rstart
	rend = rstart.Add(time.Hour * 100)
	l.Rights.End = &rend
	rprint = int32(200)
	l.Rights.Print = &rprint
	rcopy = int32(2000)
	l.Rights.Copy = &rcopy

	err = st.UpdateRights(l)
	if err != nil {
		t.Fatal(err)
	}
	l2, err = st.Get(l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *l2.Rights.Print != *l.Rights.Print {
		t.Errorf("Failed getting updated print right")
	}

	// update
	l.Provider = "him.org"
	err = st.Update(l)
	if err != nil {
		t.Fatal(err)
	}

	// update the status (revoke)
	err = st.UpdateLsdStatus(l.ID, int32(2))
	if err != nil {
		t.Fatal(err)
	}

}
