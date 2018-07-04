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
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type (
	LicensesStatusCollection []*LicenseStatus

	// just for legacy of old API
	LicenseStatusUpdatedJSON struct {
		LicenseUpdated time.Time `json:"-" gorm:"column:license" sql:"NOT NULL"`
		StatusUpdated  time.Time `json:"-" gorm:"column:status" sql:"NOT NULL"`
	}

	LicenseStatus struct {
		Id                 int64                       `json:"-" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		LicenseRef         string                      `json:"id" gorm:"column:license_ref;associationForeignKey:Id;size:36"` // uuid - max 36
		Status             Status                      `json:"status" gorm:"type:int" sql:"NOT NULL"`
		LicenseUpdated     time.Time                   `json:"-" gorm:"column:license_updated" sql:"NOT NULL"`
		StatusUpdated      time.Time                   `json:"-" gorm:"column:status_updated" sql:"NOT NULL"`
		Updated            LicenseStatusUpdatedJSON    `json:"updated" gorm:"-"` // just for legacy of old API
		DeviceCount        *NullInt                    `json:"device_count,omitempty" gorm:"column:device_count" sql:"DEFAULT NULL"`
		PotentialRightsEnd *NullTime                   `json:"potential_rights_end,omitempty" sql:"DEFAULT NULL"`
		CurrentEndLicense  *NullTime                   `json:"-" gorm:"column:rights_end" sql:"DEFAULT NULL"`
		Links              LicenseLinksCollection      `json:"links,omitempty"`
		Events             TransactionEventsCollection `json:"events,omitempty"`
		UpdatedAt          time.Time                   `json:"-"`
	}

	Status string
)

// List of status values as strings
const (
	StatusReady     Status = "Ready"
	StatusActive    Status = "Active"
	StatusRevoked   Status = "Revoked"
	StatusReturned  Status = "Returned"
	StatusCancelled Status = "Cancelled"
	StatusExpired   Status = "Expired"
	EventRenewed    Status = "Renewed"
)

// List of status values as int
const (
	StatusReadyInt     int64 = 1
	StatusActiveInt    int64 = 2
	StatusRevokedInt   int64 = 4
	StatusReturnedInt  int64 = 8
	StatusCancelledInt int64 = 16
	StatusExpiredInt   int64 = 32
	EventRenewedInt    int64 = 64
)

// Implementation of Stringer
func (t Status) String() string {
	return string(t)
}

// Implementation of sql Scanner
func (t *Status) Scan(i interface{}) error {
	var vv int64
	switch v := i.(type) {
	case nil:
		return nil
	case int64:
		vv = v
	case []byte:
		inter, err := strconv.Atoi(string(v))
		if err != nil {
			return fmt.Errorf("can't scan %T into %T (%s). Convert failed : %v\n", v, t, v, err)
		}
		vv = int64(inter)
	default:
		return fmt.Errorf("can't scan %T into %T (%#v)\n", v, t, i)
	}

	switch vv {
	case StatusReadyInt:
		*t = StatusReady
	case StatusActiveInt:
		*t = StatusActive
	case StatusRevokedInt:
		*t = StatusRevoked
	case StatusReturnedInt:
		*t = StatusReturned
	case StatusCancelledInt:
		*t = StatusCancelled
	case StatusExpiredInt:
		*t = StatusExpired
	case EventRenewedInt:
		*t = EventRenewed
	default:
		return fmt.Errorf("invalid value of type RideStatus: %v", *t)
	}
	return nil
}

// Implementation of sql Valuer
func (t Status) Value() (driver.Value, error) {
	if t == "" {
		return nil, nil
	}
	switch t {
	case StatusReady:
		return StatusReadyInt, nil
	case StatusActive:
		return StatusActiveInt, nil
	case StatusRevoked:
		return StatusRevokedInt, nil
	case StatusReturned:
		return StatusReturnedInt, nil
	case StatusCancelled:
		return StatusCancelledInt, nil
	case StatusExpired:
		return StatusExpiredInt, nil
	case EventRenewed:
		return EventRenewedInt, nil
	default:
		return nil, fmt.Errorf("invalid value of type RideStatus: %v", t)
	}
}

func (s *LicenseStatus) String() string {
	result := fmt.Sprintf("%d Status : %s LicRef : %s\n", s.Id, s.Status, s.LicenseRef)
	result += " Updated on : " + s.LicenseUpdated.String()
	result += " Status Updated on : " + s.StatusUpdated.String()
	if s.DeviceCount != nil && s.DeviceCount.Valid {
		result += fmt.Sprintf(" %d devices", s.DeviceCount.Int64)
	}
	if s.PotentialRightsEnd != nil && s.PotentialRightsEnd.Valid {
		result += " PotentialRightsEnd : " + s.PotentialRightsEnd.Time.String()
	}
	if s.CurrentEndLicense != nil && s.CurrentEndLicense.Valid {
		result += " CurrentEndLicense : " + s.CurrentEndLicense.Time.String()
	}
	result += fmt.Sprintf(" %d links %d events", len(s.Links), len(s.Events))
	return result
}

func (s *LicenseStatus) TableName() string {
	return LSDLicenseStatusTableName
}

// Implementation of GORM callback
func (s *LicenseStatus) BeforeSave() error {
	return nil
}

// Implementation of GORM callback
func (s *LicenseStatus) AfterFind() error {
	// clear device count if not valid
	if s.DeviceCount != nil {
		if !s.DeviceCount.Valid {
			s.DeviceCount = nil
		}
	}
	// just for legacy of old API
	s.Updated = LicenseStatusUpdatedJSON{
		StatusUpdated:  s.StatusUpdated,
		LicenseUpdated: s.LicenseUpdated,
	}
	return nil
}

//Add adds license status to database
func (i licenseStatusStore) Add(ls *LicenseStatus) error {
	return i.db.Create(ls).Error
}

//Update updates license status
func (i licenseStatusStore) Update(ls *LicenseStatus) error {
	return i.db.Save(ls).Error
}

// Counts statuses which have devices count more than devices limit
func (i licenseStatusStore) Count(deviceLimit int64) (int64, error) {
	var result int64
	//`SELECT COUNT(*) FROM license_status WHERE device_count >= ?
	return result, i.db.Model(LicenseStatus{}).Where("device_count >= ?", deviceLimit).Count(&result).Error
}

//List gets license statuses which have devices count more than devices limit
//input parameters: limit - how much license statuses need to get, offset - from what position need to start
func (i licenseStatusStore) List(deviceLimit int64, page int64, pageNum int64) (LicensesStatusCollection, error) {
	var result LicensesStatusCollection
	//`SELECT status, license_updated, status_updated, device_count, license_ref FROM license_status WHERE device_count >= ? ORDER BY id DESC LIMIT ? OFFSET ?`
	return result, i.db.Where("device_count >= ?", deviceLimit).Offset(pageNum * page).Limit(page).Order("id DESC").Find(&result).Error
}

func (i licenseStatusStore) ListLatest(timestamp time.Time) (LicensesStatusCollection, error) {
	var result LicensesStatusCollection
	return result, i.db.Where("updated_at > ?", timestamp).Order("id DESC").Find(&result).Error
}

func (i licenseStatusStore) ListAll() (LicensesStatusCollection, error) {
	var result LicensesStatusCollection
	return result, i.db.Order("id DESC").Find(&result).Error
}

// GetByLicenseId gets license status by license id
func (i licenseStatusStore) GetByLicenseId(licenseFk string) (*LicenseStatus, error) {
	var result LicenseStatus
	return &result, i.db.Model(LicenseStatus{}).Where("license_ref = ?", licenseFk).Find(&result).Error
}

// DeleteByLicenseIds deletes all statuses for a license_ref. licenseFks is comma separated array of ids
func (i licenseStatusStore) DeleteByLicenseIds(licenseFks string) error {
	fks := strings.Split(licenseFks, ",")
	return i.db.Where("license_ref IN (?)", fks).Delete(LicenseStatus{}).Error
}
