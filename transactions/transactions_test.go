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

func TestCRUD(t *testing.T) {

	config.Config.LsdServer.Database = "sqlite3://:memory:"
	driver, cnxn := config.GetDatabase(config.Config.LsdServer.Database)
	db, err := sql.Open(driver, cnxn)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := time.Now().UTC().Truncate(time.Second)

	// add
	e := Event{DeviceName: "testdevice", Timestamp: timestamp, DeviceId: "deviceid1", LicenseStatusFk: 1}
	err = evt.Add(e, status.STATUS_REVOKED_INT)
	if err != nil {
		t.Error(err)
	}

	// get
	event, err := evt.Get(1)
	if err != nil {
		t.Error(err)
	}
	if event.ID != 1 {
		t.Errorf("Failed getting item 1, got %d instead", event.ID)
	}
	if event.Type != "revoke" {
		t.Errorf("Failed getting type revoke, got %s instead", event.Type)
	}

	// get by license status id
	fn := evt.GetByLicenseStatusId(1)
	eventList := make([]Event, 0)
	for it, err := fn(); err == nil; it, err = fn() {
		eventList = append(eventList, it)
	}
	if len(eventList) != 1 {
		t.Errorf("Failed getting a list with one item, got %d instead", len(eventList))
	}
	if eventList[0].Type != "revoke" {
		t.Errorf("Failed getting type revoke, got %s instead", eventList[0].Type)
	}

	// add more
	timestamp = time.Now().UTC().Truncate(time.Second)
	e = Event{DeviceName: "testdevice", Timestamp: timestamp, DeviceId: "deviceid2", LicenseStatusFk: 1}
	err = evt.Add(e, status.STATUS_ACTIVE_INT)
	if err != nil {
		t.Error(err)
	}
	e = Event{DeviceName: "testdevice", Timestamp: timestamp, DeviceId: "deviceid3", LicenseStatusFk: 1}
	err = evt.Add(e, status.STATUS_ACTIVE_INT)
	if err != nil {
		t.Error(err)
	}

	// list registered devices
	fnr := evt.ListRegisteredDevices(1)
	deviceList := make([]Device, 0)
	for it, err := fnr(); err == nil; it, err = fnr() {
		deviceList = append(deviceList, it)
	}
	if len(deviceList) != 2 {
		t.Errorf("Failed getting a list with two items, got %d instead", len(eventList))
	}
	if deviceList[0].DeviceId != "deviceid2" {
		t.Errorf("Failed getting a proper deviceid, got %s instead", deviceList[0].DeviceId)
	}

	// check device status
	dstat, err := evt.CheckDeviceStatus(1, "deviceid2")
	if err != nil {
		t.Error(err)
	}
	if dstat != "register" {
		t.Errorf("Failed getting a proper device status, got %s instead", dstat)
	}

	dstat, err = evt.CheckDeviceStatus(1, "deviceid1")
	if err != nil {
		t.Error(err)
	}
	if dstat != "revoke" {
		t.Errorf("Failed getting a proper device status, got %s instead", dstat)
	}

}
