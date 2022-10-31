// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package licensestatuses

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/readium/readium-lcp-server/config"
)

func TestCRUD(t *testing.T) {

	config.Config.LsdServer.Database = "sqlite3://:memory:"
	driver, cnxn := config.GetDatabase(config.Config.LsdServer.Database)
	db, err := sql.Open(driver, cnxn)
	if err != nil {
		t.Fatal(err)
	}

	lst, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := time.Now().UTC().Truncate(time.Second)

	// add
	count := 2
	ls := LicenseStatus{PotentialRights: &PotentialRights{End: &timestamp}, LicenseRef: "licenseref", Status: "active", Updated: &Updated{License: &timestamp, Status: &timestamp}, DeviceCount: &count}
	err = lst.Add(ls)
	if err != nil {
		t.Error(err)
	}

	// list with no device limit, 10 max, offset 0
	fn := lst.List(0, 10, 0)
	if fn == nil {
		t.Errorf("Failed getting a non null list function")
	}
	statusList := make([]LicenseStatus, 0)
	for it, err := fn(); err == nil; it, err = fn() {
		statusList = append(statusList, it)
	}
	if len(statusList) != 1 {
		t.Errorf("Failed getting a list with one item, got %d instead", len(statusList))
	}

	// get by id
	statusID := statusList[0].ID
	_, err = lst.GetByID(statusID)
	if err != nil {
		t.Error(err)
	}

	// get by license id
	ls2, err := lst.GetByLicenseID("licenseref")
	if err != nil {
		t.Error(err)
	}
	if ls2.ID != statusID {
		t.Errorf("Failed getting a license status by license id")
	}

	// update
	ls2.Status = "revoked"
	err = lst.Update(*ls2)
	if err != nil {
		t.Error(err)
	}

	ls3, err := lst.GetByID(ls2.ID)
	if err != nil {
		t.Error(err)
	}
	if ls3.Status != "revoked" {
		t.Errorf("Failed getting the proper stats, got %s instead", ls3.Status)
	}

}
