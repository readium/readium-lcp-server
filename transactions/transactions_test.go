package transactions

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/readium/readium-lcp-server/status"
)

//TestTransactionCreation opens database and tries to add an event to table 'event'
func TestTransactionCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	trns, err := Open(db)
	if err != nil {
		t.Error("Can't open transactions")
		t.Error(err)
		t.FailNow()
	}

	timestamp := time.Now()

	e := Event{DeviceName: "testdevice", Timestamp: timestamp, Type: status.Types[1], DeviceId: "deviceid", LicenseStatusFk: 1}
	err = trns.Add(e, 1)
	if err != nil {
		t.Error(err)
	}
	_, err = trns.Get(1)
	if err != nil {
		t.Error(err)
	}
}
