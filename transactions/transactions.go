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

package transactions

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/readium/readium-lcp-server/status"
)

var NotFound = errors.New("Event not found")

type Transactions interface {
	Get(id int) (Event, error)
	Add(e Event, eventType int) error
	GetByLicenseStatusId(licenseStatusFk int) func() (Event, error)
	CheckDeviceStatus(licenseStatusFk int, deviceId string) (string, error)
	ListRegisteredDevices(licenseStatusFk int) func() (Device, error)
}

type RegisteredDevicesList struct {
	Id      string   `json:"id"`
	Devices []Device `json:"devices"`
}

type Device struct {
	DeviceId   string    `json:"id"`
	DeviceName string    `json:"name"`
	Timestamp  time.Time `json:"timestamp"`
}

type Event struct {
	Id              int       `json:"-"`
	DeviceName      string    `json:"name"`
	Timestamp       time.Time `json:"timestamp"`
	Type            string    `json:"type"`
	DeviceId        string    `json:"id"`
	LicenseStatusFk int       `json:"-"`
}

type dbTransactions struct {
	db                    *sql.DB
	get                   *sql.Stmt
	add                   *sql.Stmt
	getbylicensestatusid  *sql.Stmt
	checkdevicestatus     *sql.Stmt
	listregistereddevices *sql.Stmt
}

// Get returns an event by its id
//
func (i dbTransactions) Get(id int) (Event, error) {
	records, err := i.get.Query(id)
	var typeInt int

	defer records.Close()
	if records.Next() {
		var e Event
		err = records.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &typeInt, &e.DeviceId, &e.LicenseStatusFk)
		if err == nil {
			e.Type = status.EventTypes[typeInt]
		}
		return e, err
	}

	return Event{}, NotFound
}

// Add adds an event in the database,
// The parameter eventType corresponds to the field 'type' in table 'event'
//
func (i dbTransactions) Add(e Event, eventType int) error {
	add, err := i.db.Prepare("INSERT INTO event (device_name, timestamp, type, device_id, license_status_fk) VALUES (?, ?, ?, ?, ?)")

	if err != nil {
		return err
	}

	defer add.Close()
	_, err = add.Exec(e.DeviceName, e.Timestamp, eventType, e.DeviceId, e.LicenseStatusFk)
	return err
}

// GetByLicenseStatusId returns all events by license status id
//
func (i dbTransactions) GetByLicenseStatusId(licenseStatusFk int) func() (Event, error) {
	rows, err := i.getbylicensestatusid.Query(licenseStatusFk)
	if err != nil {
		return func() (Event, error) { return Event{}, err }
	}
	return func() (Event, error) {
		var e Event
		var err error
		var typeInt int

		if rows.Next() {
			err = rows.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &typeInt, &e.DeviceId, &e.LicenseStatusFk)
			if err == nil {
				e.Type = status.EventTypes[typeInt]
			}
		} else {
			rows.Close()
			err = NotFound
		}
		return e, err
	}
}

// ListRegisteredDevices returns all devices which have an 'active' status by licensestatus id
//
func (i dbTransactions) ListRegisteredDevices(licenseStatusFk int) func() (Device, error) {
	rows, err := i.listregistereddevices.Query(licenseStatusFk)
	if err != nil {
		return func() (Device, error) { return Device{}, err }
	}
	return func() (Device, error) {
		var d Device
		var err error
		if rows.Next() {
			err = rows.Scan(&d.DeviceId, &d.DeviceName, &d.Timestamp)
		} else {
			rows.Close()
			err = NotFound
		}
		return d, err
	}
}

// CheckDeviceStatus gets the current status of a device
// if the device has not been recorded in the 'event' table, typeString is empty.
//
func (i dbTransactions) CheckDeviceStatus(licenseStatusFk int, deviceId string) (string, error) {
	var typeString string
	var typeInt int

	row := i.checkdevicestatus.QueryRow(licenseStatusFk, deviceId)
	err := row.Scan(&typeInt)

	if err == nil {
		typeString = status.EventTypes[typeInt]
	} else {
		if err == sql.ErrNoRows {
			return typeString, nil
		}
	}

	return typeString, err
}

// Open defines scripts for queries & create the 'event' table if it does not exist
//
func Open(db *sql.DB) (t Transactions, err error) {
	// create the event table if it does not exist
	_, err = db.Exec(tableDef)
	if err != nil {
		log.Println("Error creating event table")
		return
	}

	// select an event by its id
	get, err := db.Prepare("SELECT * FROM event WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}

	getbylicensestatusid, err := db.Prepare("SELECT * FROM event WHERE license_status_fk = ?")

	// the status of a device corresponds to the latest event stored in the db.
	checkdevicestatus, err := db.Prepare(`SELECT type FROM event WHERE license_status_fk = ?
	AND device_id = ? ORDER BY timestamp DESC LIMIT 1`)

	listregistereddevices, err := db.Prepare(`SELECT device_id,
	device_name, timestamp  FROM event  WHERE license_status_fk = ? AND type = 1`)

	if err != nil {
		return
	}

	t = dbTransactions{db, get, nil, getbylicensestatusid, checkdevicestatus, listregistereddevices}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS `event` (" +
	"id integer PRIMARY KEY," +
	"device_name varchar(255) DEFAULT NULL," +
	"`timestamp` datetime NOT NULL," +
	"`type` int NOT NULL," +
	"device_id varchar(255) DEFAULT NULL," +
	"license_status_fk int NOT NULL," +
	"FOREIGN KEY(license_status_fk) REFERENCES license_status(id)" +
	");" +
	"CREATE INDEX license_status_fk_index on event (license_status_fk);"
