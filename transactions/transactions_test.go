// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package transactions

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/status"
)

//TestTransactionCreation opens database and tries to add an event to table 'event'
func TestTransactionCreation(t *testing.T) {
	config.Config.LsdServer.Database = "sqlite" // FIXME

	db, err := sql.Open("sqlite3", ":memory:")
	trns, err := Open(db)
	if err != nil {
		t.Error("Can't open transactions")
		t.Error(err)
		t.FailNow()
	}

	timestamp := time.Now().UTC().Truncate(time.Second)

	e := Event{DeviceName: "testdevice", Timestamp: timestamp, Type: status.EventTypes[1], DeviceId: "deviceid", LicenseStatusFk: 1}
	err = trns.Add(e, 1)
	if err != nil {
		t.Error(err)
	}
	_, err = trns.Get(1)
	if err != nil {
		t.Error(err)
	}
}
