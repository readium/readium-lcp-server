// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package transactions

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/dbutils"
	"github.com/readium/readium-lcp-server/status"
)

var ErrNotFound = errors.New("Event not found")

type Transactions interface {
	Get(id int) (Event, error)
	Add(e Event, eventType int) error
	GetByLicenseStatusId(licenseStatusFk int) func() (Event, error)
	CheckDeviceStatus(licenseStatusFk int, deviceId string) (string, error)
	ListRegisteredDevices(licenseStatusFk int) func() (Device, error)
}

type RegisteredDevicesList struct {
	ID      string   `json:"id"`
	Devices []Device `json:"devices"`
}

type Device struct {
	DeviceId   string    `json:"id"`
	DeviceName string    `json:"name"`
	Timestamp  time.Time `json:"timestamp"`
}

type Event struct {
	ID              int       `json:"-"`
	DeviceName      string    `json:"name"`
	Timestamp       time.Time `json:"timestamp"`
	Type            string    `json:"type"`
	DeviceId        string    `json:"id"`
	LicenseStatusFk int       `json:"-"`
}

type dbTransactions struct {
	db                      *sql.DB
	dbGet                   *sql.Stmt
	dbGetByStatusID         *sql.Stmt
	dbCheckDeviceStatus     *sql.Stmt
	dbListRegisteredDevices *sql.Stmt
}

// Get returns an event by its id
func (i dbTransactions) Get(id int) (Event, error) {

	row := i.dbGet.QueryRow(id)
	var e Event
	var typeInt int
	err := row.Scan(&e.ID, &e.DeviceName, &e.Timestamp, &typeInt, &e.DeviceId, &e.LicenseStatusFk)
	if err != nil {
		return Event{}, err
	}
	e.Type = status.EventTypes[typeInt]
	return e, err
}

// Add adds an event in the database,
// The parameter eventType corresponds to the field 'type' in table 'event'
func (i dbTransactions) Add(e Event, eventType int) error {

	_, err := i.db.Exec(dbutils.GetParamQuery(config.Config.LsdServer.Database, "INSERT INTO event (device_name, timestamp, type, device_id, license_status_fk) VALUES (?, ?, ?, ?, ?)"),
		e.DeviceName, e.Timestamp, eventType, e.DeviceId, e.LicenseStatusFk)
	return err
}

// GetByLicenseStatusId returns all events by license status id
func (i dbTransactions) GetByLicenseStatusId(licenseStatusFk int) func() (Event, error) {
	rows, err := i.dbGetByStatusID.Query(licenseStatusFk)
	if err != nil {
		return func() (Event, error) { return Event{}, err }
	}
	return func() (Event, error) {
		var e Event
		var err error
		var typeInt int

		if rows.Next() {
			err = rows.Scan(&e.ID, &e.DeviceName, &e.Timestamp, &typeInt, &e.DeviceId, &e.LicenseStatusFk)
			if err == nil {
				e.Type = status.EventTypes[typeInt]
			}
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return e, err
	}
}

// ListRegisteredDevices returns all devices which have an 'active' status by licensestatus id
func (i dbTransactions) ListRegisteredDevices(licenseStatusFk int) func() (Device, error) {

	rows, err := i.dbListRegisteredDevices.Query(licenseStatusFk)
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
			err = ErrNotFound
		}
		return d, err
	}
}

// CheckDeviceStatus gets the current status of a device as a string
// if the device has not been recorded in the 'event' table, returns an empty string.
func (i dbTransactions) CheckDeviceStatus(licenseStatusFk int, deviceId string) (string, error) {
	var typeString string
	var typeInt int

	row := i.dbCheckDeviceStatus.QueryRow(licenseStatusFk, deviceId)
	err := row.Scan(&typeInt)
	if err != nil {
		if err == sql.ErrNoRows {
			return typeString, nil
		}
	}
	typeString = status.EventTypes[typeInt]
	return typeString, err
}

// Open defines scripts for queries & create the 'event' table if it does not exist
func Open(db *sql.DB) (t Transactions, err error) {

	driver, _ := config.GetDatabase(config.Config.LsdServer.Database)

	// if sqlite, create the event table in the lsd db if it does not exist
	if driver == "sqlite3" {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating sqlite event table")
			return
		}
	}

	// select an event by its id
	dbGet, err := db.Prepare(dbutils.GetParamQuery(config.Config.LsdServer.Database, "SELECT * FROM event WHERE id = ?"))
	if err != nil {
		return
	}

	dbGetByStatusID, err := db.Prepare(dbutils.GetParamQuery(config.Config.LsdServer.Database, "SELECT * FROM event WHERE license_status_fk = ?"))
	if err != nil {
		return
	}

	// the status of a device corresponds to the latest event stored in the db.
	var dbCheckDeviceStatus *sql.Stmt
	if driver == "mssql" {
		dbCheckDeviceStatus, err = db.Prepare(`SELECT TOP 1 type FROM event WHERE license_status_fk = ?
		AND device_id = ? ORDER BY timestamp DESC`)
	} else {
		dbCheckDeviceStatus, err = db.Prepare(dbutils.GetParamQuery(config.Config.LsdServer.Database, `SELECT type FROM event WHERE license_status_fk = ?
		AND device_id = ? ORDER BY timestamp DESC LIMIT 1`))
	}
	if err != nil {
		return
	}

	dbListRegisteredDevices, err := db.Prepare(dbutils.GetParamQuery(config.Config.LsdServer.Database, `SELECT device_id,
	device_name, timestamp  FROM event  WHERE license_status_fk = ? AND type = 1`))
	if err != nil {
		return
	}

	t = dbTransactions{db, dbGet, dbGetByStatusID, dbCheckDeviceStatus, dbListRegisteredDevices}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS event (" +
	"id integer PRIMARY KEY," +
	"device_name varchar(255) DEFAULT NULL," +
	"timestamp datetime NOT NULL," +
	"type int NOT NULL," +
	"device_id varchar(255) DEFAULT NULL," +
	"license_status_fk int NOT NULL," +
	"FOREIGN KEY(license_status_fk) REFERENCES license_status(id)" +
	");" +
	"CREATE INDEX IF NOT EXISTS license_status_fk_index on event (license_status_fk);"
