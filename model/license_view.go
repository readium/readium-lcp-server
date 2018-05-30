/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package model

import "github.com/jinzhu/gorm"

type (
	// License struct defines a license
	LicenseView struct {
		ID               int      `json:"-" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		PublicationTitle string   `json:"publication_title" gorm:"-"`
		UserName         string   `json:"user_name" gorm:"-"`
		Type             string   `json:"type" gorm:"-"`
		UUID             string   `json:"id" gorm:"size:36"` //uuid - max size 36
		DeviceCount      *NullInt `json:"device_count" sql:"NOT NULL"`
		Status           Status   `json:"status"  sql:"NOT NULL"`
		PurchaseID       int      `json:"purchase_id" gorm:"-"`
		Message          string   `json:"message" sql:"NOT NULL"`
	}

	LicensesViewCollection []*LicenseView
)

// Implementation of gorm Tabler
func (l *LicenseView) TableName() string {
	return LUTLicenseViewTableName
}

// Get a license for a given ID
//
func (s licenseStore) GetView(id int64) (License, error) {
	/**
		dbGetByID, err := s.db.Prepare(`SELECT l.uuid, pu.title, u.name, p.type, l.device_count, l.status, p.id, l.message FROM license_view AS l
												INNER JOIN purchase as p ON l.uuid = p.license_uuid
												INNER JOIN publication as pu ON p.publication_id = pu.id
												INNER JOIN user as u ON p.user_id = u.id
												WHERE id = ?`)
		if err != nil {
			return LicenseView{}, err
		}
		defer dbGetByID.Close()

		records, err := dbGetByID.Query(id)
		if records.Next() {
			var result LicenseView
			err = records.Scan(
				&result.ID,
				&result.PublicationTitle,
				&result.UserName,
				&result.Type,
				&result.DeviceCount,
				&result.Status,
				&result.PurchaseID,
				&result.Message)
			records.Close()
			return result, err
		}
	**/
	return License{}, gorm.ErrRecordNotFound
}

// GetFiltered give a license with more than the filtered number
//
func (s licenseStore) GetFiltered(filter string) (LicensesCollection, error) {
	/**
	dbGetByID, err := s.db.Prepare(`SELECT l.uuid, pu.title, u.name, p.type, l.device_count, l.status, p.id, l.message FROM license_view AS l
											INNER JOIN purchase as p ON l.uuid = p.license_uuid
											INNER JOIN publication as pu ON p.publication_id = pu.id
											INNER JOIN user as u ON p.user_id = u.id
											WHERE l.device_count >= ?`)
	if err != nil {
		return []LicenseView{}, err
	}
	defer dbGetByID.Close()
	records, err := dbGetByID.Query(filter)
	result := make([]LicenseView, 0, 20)

	for records.Next() {
		var lic LicenseView
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
		result = append(result, lic)
	}
	records.Close()

	return result, nil
	**/
	return make(LicensesCollection, 0, 0), nil
}

// Add adds a new license
//
func (s licenseStore) AddView(licenses LicenseView) error {
	return s.db.Create(licenses).Error
}

// AddFromJSON adds a new license from a JSON string
//
func (s licenseStore) BulkAdd(licenses LicensesStatusCollection) error {
	result := Transaction(s.db, func(tx txStore) error {
		for _, l := range licenses {
			entity := &LicenseView{
				UUID:        l.LicenseRef,
				DeviceCount: l.DeviceCount,
				Status:      l.Status,
			}
			err := tx.Create(entity).Error
			if err != nil {
				return err
			}
		}
		return nil
	})

	return result
}

// PurgeDataBase erases all the content of the license_view table
//
func (s licenseStore) PurgeDataBase() error {
	return s.db.Delete(LicenseView{}).Error
}

// Update updates a license
//
func (s licenseStore) UpdateView(lic LicenseView) error {
	return s.db.Save(lic).Error
}

// Delete deletes a license
//
func (s licenseStore) Delete(id int64) error {
	return s.db.Where("id = ?", id).Delete(LicenseView{}).Error
}
