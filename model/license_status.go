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
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

type (
	LicensesStatusCollection []*LicenseStatus

	LicenseStatus struct {
		Id                 int64                       `json:"-" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		Status             Status                      `json:"status" gorm:"type:int"`
		LicenseUpdated     *NullTime                   `json:"license,omitempty" gorm:"column:license_updated"`
		StatusUpdated      *NullTime                   `json:"status,omitempty" gorm:"column:status_updated"`
		DeviceCount        *NullInt                    `json:"device_count,omitempty" gorm:"column:device_count"`
		PotentialRightsEnd *NullTime                   `json:"potential_rights,omitempty"`
		LicenseRef         string                      `json:"id" gorm:"column:license_ref;associationForeignKey:Id"`
		CurrentEndLicense  *NullTime                   `json:"-" gorm:"column:rights_end"`
		Links              LicenseLinksCollection      `json:"links,omitempty"`
		Events             TransactionEventsCollection `json:"events,omitempty"`
		//Message            string                      `json:"message"` // TODO : this was never completed ? there was no write into it, just read
	}

	Status string
)

// List of status values as strings
const (
	StatusReady     Status = "ready"
	StatusActive    Status = "active"
	StatusRevoked   Status = "revoked"
	StatusReturned  Status = "returned"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
	EventRenewed    Status = "renewed"
)

// List of status values as int
const (
	StatusReadyInt     int64 = 0
	StatusActiveInt    int64 = 1
	StatusRevokedInt   int64 = 2
	StatusReturnedInt  int64 = 3
	StatusCancelledInt int64 = 4
	StatusExpiredInt   int64 = 5
	EventRenewedInt    int64 = 6
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
	default:
		return fmt.Errorf("can't scan %T into %T", v, t)
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

func (s *LicenseStatus) TableName() string {
	return LSDLicenseStatusTableName
}

// makeLicenseStatus sets fields of license status according to the config file
// and creates needed inner objects of license status
//
func (s *LicenseStatus) MakeLicenseStatus(license *License, registerAvailable bool, rentingDays int) {
	s.LicenseRef = license.Id

	if license.Rights == nil || !license.Rights.End.Valid {
		// The publication was purchased (not a loan), so we do not set LSD.PotentialRights.End
		s.CurrentEndLicense.Valid = false
	} else {
		// license.Rights.End exists => this is a loan
		endFromLicense := license.Rights.End.Time.Add(0)
		s.CurrentEndLicense = &NullTime{Time: endFromLicense, Valid: true}
		if rentingDays > 0 {
			endFromConfig := license.Issued.Add(time.Hour * 24 * time.Duration(rentingDays))
			if endFromLicense.After(endFromConfig) {
				s.PotentialRightsEnd = &NullTime{Time: endFromLicense, Valid: true}
			} else {
				s.PotentialRightsEnd = &NullTime{Time: endFromConfig, Valid: true}
			}
		} else {
			s.PotentialRightsEnd = &NullTime{Time: endFromLicense, Valid: true}
		}
	}

	if registerAvailable {
		s.Status = StatusReady
	} else {
		s.Status = StatusActive
	}

	s.LicenseUpdated = &NullTime{Time: license.Issued, Valid: true}
	s.StatusUpdated = TruncatedNow()

	s.DeviceCount = &NullInt{sql.NullInt64{Int64: 0, Valid: true}}
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
	return result, i.db.Where("device_count >= ?", deviceLimit).Count(&result).Error
}

//List gets license statuses which have devices count more than devices limit
//input parameters: limit - how much license statuses need to get, offset - from what position need to start
func (i licenseStatusStore) List(deviceLimit int64, page int64, pageNum int64) (LicensesStatusCollection, error) {
	var result LicensesStatusCollection
	//`SELECT status, license_updated, status_updated, device_count, license_ref FROM license_status WHERE device_count >= ? ORDER BY id DESC LIMIT ? OFFSET ?`
	return result, i.db.Where("device_count >= ?", deviceLimit).Offset(pageNum * page).Limit(page).Order("id DESC").Find(&result).Error
}

//GetByLicenseId gets license status by license id
func (i licenseStatusStore) GetByLicenseId(licenseFk string) (*LicenseStatus, error) {
	var result LicenseStatus
	return &result, i.db.Model(LicenseStatus{}).Where("license_ref = ?", licenseFk).Find(&result).Error
}
