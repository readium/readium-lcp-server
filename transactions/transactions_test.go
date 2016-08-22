package transactions

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestTransactionCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	trns, err := Open(db)
	if err != nil {
		t.Error("Can't open transactions")
		t.Error(err)
		t.FailNow()
	}

	timestamp := time.Now()

	e := Event{DeviceName: "testdevice", Timestamp: timestamp, Type: 1, DeviceId: "deviceid", LicenseStatusFk: 1}
	err = trns.Add(e)
	if err != nil {
		t.Error(err)
	}
	_, err = trns.Get(1)
	if err != nil {
		t.Error(err)
	}
}
