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

package weblicense

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/readium/readium-lcp-server/config"
)

// License status
const (
	StatusDraft      string = "draft"
	StatusEncrypting string = "encrypting"
	StatusError      string = "error"
	StatusOk         string = "ok"
)

// ErrNotFound error trown when license is not found
var ErrNotFound = errors.New("License not found")

// WebLicense interface for license db interaction
type WebLicense interface {
	Get(id int64) (License, error)
	GetFiltered(filter string) ([]License, error)
	Add(license License) error
	AddFromJSON(licensesJSON []byte) error
	PurgeDataBase() error
	Update(license License) error
	Delete(id int64) error
}

// License struct defines a license
type License struct {
	ID               string `json:""`
	PublicationTitle string `json:"publication_title"`
	UserName         string `json:"user_name"`
	Type             string `json:"type"`
	UUID             string `json:"id"`
	DeviceCount      int    `json:"device_count"`
	Status           string `json:"status"`
	PurchaseID       int    `json:"purchase_id"`
	Message          string `json:"message"`
}

// Licenses struct defines a licenses array to be transfered
type Licenses []struct {
	ID          string `json:""`
	UUID        string `json:"id"`
	Status      string `json:"status"`
	DeviceCount int    `json:"device_count"`
	Message     string `json:"message"`
}

// LicenseManager helper
type LicenseManager struct {
	config config.Configuration
	db     *sql.DB
}

// Get a license for a given ID
//
func (licManager LicenseManager) Get(id int64) (License, error) {
	dbGetByID, err := licManager.db.Prepare(`SELECT l.uuid, pu.title, u.name, p.type, l.device_count, l.status, p.id, l.message FROM license AS l 
											INNER JOIN purchase as p ON l.uuid = p.license_uuid 
											INNER JOIN publication as pu ON p.publication_id = pu.id
											INNER JOIN user as u ON p.user_id = u.id
											WHERE id = ?`)
	if err != nil {
		return License{}, err
	}
	defer dbGetByID.Close()

	records, err := dbGetByID.Query(id)
	if records.Next() {
		var lic License
		err = records.Scan(
			&lic.ID,
			&lic.PublicationTitle,
			&lic.UserName,
			&lic.Type,
			&lic.DeviceCount,
			&lic.Status,
			&lic.PurchaseID,
			&lic.Message)
		records.Close()
		return lic, err
	}

	return License{}, ErrNotFound
}

// GetFiltered give a license with more than the filtered number
//
func (licManager LicenseManager) GetFiltered(filter string) ([]License, error) {
	dbGetByID, err := licManager.db.Prepare(`SELECT l.uuid, pu.title, u.name, p.type, l.device_count, l.status, p.id, l.message FROM license AS l 
											INNER JOIN purchase as p ON l.uuid = p.license_uuid 
											INNER JOIN publication as pu ON p.publication_id = pu.id
											INNER JOIN user as u ON p.user_id = u.id
											WHERE l.device_count >= ?`)
	if err != nil {
		return []License{}, err
	}
	defer dbGetByID.Close()
	records, err := dbGetByID.Query(filter)
	licences := make([]License, 0, 20)

	for records.Next() {
		var lic License
		err = records.Scan(
			&lic.ID,
			&lic.PublicationTitle,
			&lic.UserName,
			&lic.Type,
			&lic.DeviceCount,
			&lic.Status,
			&lic.PurchaseID,
			&lic.Message)
		if err != nil {
			fmt.Println(err)
		}
		licences = append(licences, lic)
	}
	records.Close()
	return licences, nil
}

// Add adds a new license
//
func (licManager LicenseManager) Add(licenses License) error {
	add, err := licManager.db.Prepare("INSERT INTO license (uuid, device_count, status, message) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()

	_, err = add.Exec(licenses.UUID, licenses.DeviceCount, licenses.Status, licenses.Message)
	if err != nil {
		return err
	}
	return nil
}

// AddFromJSON adds a new license from a JSON string
//
func (licManager LicenseManager) AddFromJSON(licensesJSON []byte) error {
	var licenses Licenses
	err := json.Unmarshal(licensesJSON, &licenses)
	if err != nil {
		return err
	}
	for _, l := range licenses {
		add, err := licManager.db.Prepare("INSERT INTO license (uuid, device_count, status, message) VALUES (?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer add.Close()

		_, err = add.Exec(l.UUID, l.DeviceCount, l.Status, l.Message)
		if err != nil {
			return err
		}
	}
	return nil
}

// PurgeDataBase erases all the content of the license table
//
func (licManager LicenseManager) PurgeDataBase() error {
	dbPurge, err := licManager.db.Prepare("DELETE FROM license")
	if err != nil {
		return err
	}
	defer dbPurge.Close()

	_, err = dbPurge.Exec()

	return err
}

// Update updates a license
//
func (licManager LicenseManager) Update(lic License) error {
	dbUpdate, err := licManager.db.Prepare("UPDATE license SET device_count=?, uuid=?, status=? , message=? WHERE id = ?")
	if err != nil {
		return err
	}
	defer dbUpdate.Close()
	_, err = dbUpdate.Exec(
		lic.DeviceCount,
		lic.Status,
		lic.UUID,
		lic.ID,
		lic.Message)
	if err != nil {
		return err
	}
	return err
}

// Delete deletes license
//
func (licManager LicenseManager) Delete(id int64) error {

	// delete license
	dbDelete, err := licManager.db.Prepare("DELETE FROM license WHERE id = ?")
	if err != nil {
		log.Println("Error creating license table")
		return err
	}
	defer dbDelete.Close()
	_, err = dbDelete.Exec(id)
	return err
}

// Init inits the license manager
//
func Init(config config.Configuration, db *sql.DB) (i WebLicense, err error) {
	_, err = db.Exec(tableDef)
	if err != nil {
		return
	}

	i = LicenseManager{config, db}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS license (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"device_count integer NOT NULL," +
	"`status` varchar(255) NOT NULL," +
	"message varchar(255) NOT NULL)"
