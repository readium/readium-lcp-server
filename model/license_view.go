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

import (
	"fmt"
	"original_gorm"
	"time"
)

type (
	// License struct defines a license
	LicenseView struct {
		ID             int       `sql:"AUTO_INCREMENT" gorm:"primary_key"`
		LSDID          int64     `sql:"NOT NULL"`
		UUID           string    `gorm:"size:36"` //uuid - max size 36 - purchase id
		DeviceCount    *NullInt  `sql:"NOT NULL"`
		Status         Status    `sql:"NOT NULL"`
		Message        string    `sql:"NOT NULL"`
		Purchase       Purchase  `gorm:"-"`
		LicenseUpdated time.Time `gorm:"column:license_updated"`
	}

	LicensesViewCollection []*LicenseView
)

// Implementation of gorm Tabler
func (l *LicenseView) TableName() string {
	return LUTLicenseViewTableName
}

func (s licenseStore) CountFiltered(deviceLimit string) (int64, error) {
	var result int64
	return result, s.db.Model(LicenseView{}).Where("device_count >= ?", deviceLimit).Count(&result).Error
}

// GetFiltered give a license with more than the filtered number
//
func (s licenseStore) GetFiltered(deviceLimit string, page, pageNum int64) (LicensesViewCollection, error) {
	var result LicensesViewCollection
	err := s.db.Where("device_count >= ?", deviceLimit).Offset(pageNum * page).Limit(page).Order("id DESC").Find(&result).Error

	for _, license := range result {
		var p Purchase
		sErr := s.db.Model(Purchase{}).Where("license_uuid = ?", license.UUID).Find(&p).Preload("User").Preload("Publication").Error
		if sErr != nil && sErr == gorm.ErrRecordNotFound {
			fmt.Printf("Purchase with UUID %q NOT FOUND.\n", license.UUID)
		}
		license.Purchase = p
	}

	return result, err
}

// Get a license for a given ID
//
func (s licenseStore) GetView(id int64) (*LicenseView, error) {
	var result LicenseView
	return &result, s.db.Where("id = ?", id).Find(&result).Error
}

// Add adds a new license
//
func (s licenseStore) AddView(licenses *LicenseView) error {
	return s.db.Create(licenses).Error
}

// BulkAddOrUpdate transforms and saves LSD LicenseStatus into LicenseView
func (s licenseStore) BulkAddOrUpdate(licenses LicensesStatusCollection) error {
	result := Transaction(s.db, func(tx txStore) error {
		for _, l := range licenses {
			entity := &LicenseView{
				UUID:           l.LicenseRef,
				DeviceCount:    l.DeviceCount,
				Status:         l.Status,
				LicenseUpdated: l.UpdatedAt,
			}
			err := tx.FirstOrCreate(entity, LicenseView{LSDID: l.Id}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})

	return result
}

func (s licenseStore) Latest() (time.Time, error) {
	rows, err := s.db.Model(LicenseView{}).Select("MAX(license_updated) as MaxTime").Rows()
	var maxTime time.Time
	if err == nil {
		var timeStr string
		for rows.Next() {
			rows.Scan(&timeStr)
			maxTime, err = time.Parse("2006-01-02 15:04:05.999999-07:00", timeStr)
		}
	}
	return maxTime, err
}

// Update updates a license
//
func (s licenseStore) UpdateView(lic *LicenseView) error {
	return s.db.Save(lic).Error
}

// Delete deletes a license
//
func (s licenseStore) Delete(id int64) error {
	return s.db.Where("id = ?", id).Delete(LicenseView{}).Error
}
