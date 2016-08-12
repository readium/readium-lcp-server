package history

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestHistoryCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	hist, err := Open(db)
	if err != nil {
		t.Error("Can't open history")
		t.Error(err)
		t.FailNow()
	}

	timestamp := time.Now()

	ls := LicenseStatus{PotentialRights: PotentialRights{End: timestamp}, LicenseRef: "licenseref", Status: "active", Updated: Updated{License: timestamp, Status: timestamp}, DeviceCount: 2}
	err = hist.Add(ls)
	if err != nil {
		t.Error(err)
	}
	_, err = hist.Get(1)
	if err != nil {
		t.Error(err)
	}
}
