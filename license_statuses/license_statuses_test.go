// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package licensestatuses

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//TestHistoryCreation opens database and tries to add(get) license status to(from) table 'licensestatus'
func TestHistoryCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	lst, err := Open(db)
	if err != nil {
		t.Error("Can't open licensestatuses")
		t.Error(err)
		t.FailNow()
	}

	timestamp := time.Now().UTC().Truncate(time.Second)

	ls := LicenseStatus{PotentialRights: &PotentialRights{End: &timestamp}, LicenseRef: "licenseref", Status: "active", Updated: &Updated{License: &timestamp, Status: &timestamp}, DeviceCount: 2}
	err = lst.Add(ls)
	if err != nil {
		t.Error(err)
	}
	_, err = lst.Get(1)
	if err != nil {
		t.Error(err)
	}
}
