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

package licensestatuses

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/status"
)

var NotFound = errors.New("License Status not found")

type LicenseStatuses interface {
	getById(id int) (*LicenseStatus, error)
	Add(ls LicenseStatus) error
	List(deviceLimit int64, limit int64, offset int64) func() (LicenseStatus, error)
	GetByLicenseId(id string) (*LicenseStatus, error)
	Update(ls LicenseStatus) error
}

type dbLicenseStatuses struct {
	db             *sql.DB
	get            *sql.Stmt
	add            *sql.Stmt
	list           *sql.Stmt
	getbylicenseid *sql.Stmt
	update         *sql.Stmt
}

//Get gets license status by id
//
// Removed in 94722fcb4a0a38bd5f765e67b0538f2042192ac4 but breaks the test,
// so putting it back as unexported.
func (i dbLicenseStatuses) getById(id int) (*LicenseStatus, error) {
	var statusDB int64
	ls := LicenseStatus{}

	var potentialRightsEnd *time.Time
	var licenseUpdate *time.Time
	var statusUpdate *time.Time

	row := i.get.QueryRow(id)
	err := row.Scan(&ls.Id, &statusDB, &licenseUpdate, &statusUpdate, &ls.DeviceCount, &potentialRightsEnd, &ls.LicenseRef, &ls.CurrentEndLicense)

	if err == nil {
		status.GetStatus(statusDB, &ls.Status)

		ls.Updated = new(Updated)

		if (potentialRightsEnd != nil) && (!(*potentialRightsEnd).IsZero()) {
			ls.PotentialRights = new(PotentialRights)
			ls.PotentialRights.End = potentialRightsEnd
		}

		if licenseUpdate != nil || statusUpdate != nil {
			ls.Updated.Status = statusUpdate
			ls.Updated.License = licenseUpdate
		}
	} else {
		if err == sql.ErrNoRows {
			return nil, NotFound
		}
	}

	return &ls, err
}

//Add adds license status to database
func (i dbLicenseStatuses) Add(ls LicenseStatus) error {
	add, err := i.db.Prepare("INSERT INTO license_status (status, license_updated, status_updated, device_count, potential_rights_end, license_ref,  rights_end) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()

	statusDB, err := status.SetStatus(ls.Status)

	if err == nil {
		var end time.Time
		if ls.PotentialRights != nil && ls.PotentialRights.End != nil && !(*ls.PotentialRights.End).IsZero() {
			end = *ls.PotentialRights.End
		}
		_, err = add.Exec(statusDB, ls.Updated.License, ls.Updated.Status, ls.DeviceCount, &end, ls.LicenseRef, ls.CurrentEndLicense)
	}

	return err
}

//List gets license statuses which have devices count more than devices limit
//input parameters: limit - how much license statuses need to get, offset - from what position need to start
func (i dbLicenseStatuses) List(deviceLimit int64, limit int64, offset int64) func() (LicenseStatus, error) {
	rows, err := i.list.Query(deviceLimit, limit, offset)
	if err != nil {
		return func() (LicenseStatus, error) { return LicenseStatus{}, err }
	}
	return func() (LicenseStatus, error) {
		var statusDB int64
		ls := LicenseStatus{}
		ls.Updated = new(Updated)

		var err error
		if rows.Next() {
			err = rows.Scan(&statusDB, &ls.Updated.License, &ls.Updated.Status, &ls.DeviceCount, &ls.LicenseRef)

			if err == nil {
				status.GetStatus(statusDB, &ls.Status)
			}
		} else {
			rows.Close()
			err = NotFound
		}
		return ls, err
	}
}

//GetByLicenseId gets license status by license id
func (i dbLicenseStatuses) GetByLicenseId(licenseFk string) (*LicenseStatus, error) {
	var statusDB int64
	ls := LicenseStatus{}

	var potentialRightsEnd *time.Time
	var licenseUpdate *time.Time
	var statusUpdate *time.Time

	row := i.getbylicenseid.QueryRow(licenseFk)
	err := row.Scan(&ls.Id, &statusDB, &licenseUpdate, &statusUpdate, &ls.DeviceCount, &potentialRightsEnd, &ls.LicenseRef, &ls.CurrentEndLicense)

	if err == nil {
		status.GetStatus(statusDB, &ls.Status)

		ls.Updated = new(Updated)

		if (potentialRightsEnd != nil) && (!(*potentialRightsEnd).IsZero()) {
			ls.PotentialRights = new(PotentialRights)
			ls.PotentialRights.End = potentialRightsEnd
		}

		if licenseUpdate != nil || statusUpdate != nil {
			ls.Updated.Status = statusUpdate
			ls.Updated.License = licenseUpdate
		}
	} else {
		if err == sql.ErrNoRows {
			return nil, err
		}
	}

	return &ls, err
}

//Update updates license status
func (i dbLicenseStatuses) Update(ls LicenseStatus) error {

	statusInt, err := status.SetStatus(ls.Status)
	if err != nil {
		return err
	}

	var potentialRightsEnd *time.Time

	if ls.PotentialRights != nil && ls.PotentialRights.End != nil && !(*ls.PotentialRights.End).IsZero() {
		potentialRightsEnd = ls.PotentialRights.End
	}

	var result sql.Result
	result, err = i.db.Exec("UPDATE license_status SET status=?, license_updated=?, status_updated=?, device_count=?,potential_rights_end=?,  rights_end=?  WHERE id=?",
		statusInt, ls.Updated.License, ls.Updated.Status, ls.DeviceCount, potentialRightsEnd, ls.CurrentEndLicense, ls.Id)

	if err == nil {
		if r, _ := result.RowsAffected(); r == 0 {
			return NotFound
		}
	}
	return err
}

//Open defines scripts for queries & create table license_status if it does not exist
func Open(db *sql.DB) (l LicenseStatuses, err error) {
	// if sqlite, create the license_status table in the lsd db if it does not exist
	if strings.HasPrefix(config.Config.LsdServer.Database, "sqlite") {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating license_status table")
			return
		}
	}

	get, err := db.Prepare("SELECT * FROM license_status WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}

	list, err := db.Prepare(`SELECT status, license_updated, status_updated, device_count, license_ref FROM license_status WHERE device_count >= ?
		ORDER BY id DESC LIMIT ? OFFSET ?`)

	getbylicenseid, err := db.Prepare("SELECT * FROM license_status where license_ref = ?")

	if err != nil {
		return
	}
	l = dbLicenseStatuses{db, get, nil, list, getbylicenseid, nil}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS license_status (" +
	"id INTEGER PRIMARY KEY," +
	"status int(11) NOT NULL," +
	"license_updated datetime NOT NULL," +
	"status_updated datetime NOT NULL," +
	"device_count int(11) DEFAULT NULL," +
	"potential_rights_end datetime DEFAULT NULL," +
	"license_ref varchar(255) NOT NULL," +
	"rights_end datetime DEFAULT NULL  " +
	");" +
	"CREATE INDEX IF NOT EXISTS license_ref_index on license_status (license_ref);"
