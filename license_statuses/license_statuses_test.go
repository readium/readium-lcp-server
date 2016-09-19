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

	timestamp := time.Now()

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
