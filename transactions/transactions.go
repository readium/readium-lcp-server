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
	"time"

	"github.com/readium/readium-lcp-server/status"
)

var NotFound = errors.New("Event not found")

type Transactions interface {
	Get(id int) (Event, error)
	Add(e Event, typeEvent int) error
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

//Get returns event if it exists in table 'event'
func (i dbTransactions) Get(id int) (Event, error) {
	records, err := i.get.Query(id)
	var typeInt int

	defer records.Close()
	if records.Next() {
		var e Event
		err = records.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &typeInt, &e.DeviceId, &e.LicenseStatusFk)
		if err == nil {
			e.Type = status.Types[typeInt]
		}
		return e, err
	}

	return Event{}, NotFound
}

//Add adds event in database, parameter typeEvent is for field 'type' in table 'event'
//1 when register device, 2 when return and 3 when renew
func (i dbTransactions) Add(e Event, typeEvent int) error {
	add, err := i.db.Prepare("INSERT INTO event (device_name, timestamp, type, device_id, license_status_fk) VALUES (?, ?, ?, ?, ?)")

	if err != nil {
		return err
	}

	defer add.Close()
	_, err = add.Exec(e.DeviceName, e.Timestamp, typeEvent, e.DeviceId, e.LicenseStatusFk)
	return err
}

//GetByLicenseStatusId returns all events by licensestatus id
func (i dbTransactions) GetByLicenseStatusId(licenseStatusFk int) func() (Event, error) {
	rows, err := i.getbylicensestatusid.Query(licenseStatusFk)
	if err != nil {
		return func() (Event, error) { return Event{}, err }
	}
	return func() (Event, error) {
		var e Event
		var err error
		if rows.Next() {
			err = rows.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &e.Type, &e.DeviceId, &e.LicenseStatusFk)
		} else {
			rows.Close()
			err = NotFound
		}
		return e, err
	}
}

//ListRegisteredDevices returns all devices which has status 'regitered' by licensestatus id
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

//CheckDeviceStatus gets current status of device
//if there is no device in table 'event' by deviceId, typeString will be the empty string
func (i dbTransactions) CheckDeviceStatus(licenseStatusFk int, deviceId string) (string, error) {
	var typeString string
	var typeInt int

	row := i.checkdevicestatus.QueryRow(licenseStatusFk, deviceId)
	err := row.Scan(&typeInt)

	if err == nil {
		typeString = status.Types[typeInt]
	} else {
		if err == sql.ErrNoRows {
			return typeString, nil
		}
	}

	return typeString, err
}

//Open defines scripts for queries & create table 'event' if not exist
func Open(db *sql.DB) (t Transactions, err error) {
	_, err = db.Exec(tableDef)
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT * FROM event WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}

	getbylicensestatusid, err := db.Prepare("SELECT * FROM event WHERE license_status_fk = ?")

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

const tableDef = `CREATE TABLE IF NOT EXISTS event (
	id INTEGER PRIMARY KEY,
	device_name varchar(255) DEFAULT NULL,
	timestamp datetime NOT NULL,
	type int NOT NULL,
	device_id varchar(255) DEFAULT NULL,
	license_status_fk int NOT NULL,
  	FOREIGN KEY(license_status_fk) REFERENCES license_status(id)
);
CREATE INDEX IF NOT EXISTS license_status_fk_index on event (license_status_fk);
`
