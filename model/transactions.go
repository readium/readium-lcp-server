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
	"github.com/satori/go.uuid"
	"time"
)

type (
	TransactionEventsCollection []*TransactionEvent

	RegisteredDevicesList struct {
		Id      string                      `json:"id"`
		Devices TransactionEventsCollection `json:"devices"`
	}

	TransactionEvent struct {
		Id              int64     `json:"-" sql:"AUTO_INCREMENT" gorm:"primary_key"`
		DeviceName      string    `json:"name"`
		Timestamp       time.Time `json:"timestamp" sql:"DEFAULT:current_timestamp;NOT NULL"`
		Type            Status    `json:"type" gorm:"type:int"`
		DeviceId        string    `json:"id" sql:"NOT NULL"`                 // TODO : is this unique?
		LicenseStatusFk int64     `json:"-" gorm:"associationForeignKey:Id"` // foreign key belongs-to License
	}
)

// Implementation of gorm Tabler
func (e *TransactionEvent) TableName() string {
	return LSDTransactionTableName
}

// Implementation of gorm callback
func (e *TransactionEvent) AfterFind() error {
	return nil
}

// Implementation of gorm callback
func (e *TransactionEvent) BeforeSave() error {
	if e.DeviceId == "" {
		// Create uuid - used in tests
		uid, errU := uuid.NewV4()
		if errU != nil {
			return errU
		}
		e.DeviceId = uid.String()
	}
	return nil
}

// Get returns an event by its id
//
func (i transactionEventStore) Get(id int64) (*TransactionEvent, error) {
	var result TransactionEvent
	return &result, i.db.Model(TransactionEvent{}).Where("id = ?", id).First(&result).Error
}

// Add adds an event in the database,
// The parameter eventType corresponds to the field 'type' in table 'event'
//
func (i transactionEventStore) Add(newTrans *TransactionEvent) error {
	return i.db.Create(newTrans).Error
}

// GetByLicenseStatusId returns all events by license status id
//
func (i transactionEventStore) GetByLicenseStatusId(licenseStatusFk int64) (TransactionEventsCollection, error) {
	var result TransactionEventsCollection
	return result, i.db.Where("license_status_fk = ?", licenseStatusFk).Find(&result).Error
}

// ListRegisteredDevices returns all devices which have an 'active' status by licensestatus id
//
func (i transactionEventStore) ListRegisteredDevices(licenseStatusFk int64) (TransactionEventsCollection, error) {
	var result TransactionEventsCollection
	//`SELECT device_id, device_name, timestamp  FROM event  WHERE license_status_fk = ? AND type = 1`
	return result, i.db.Where("%s.license_status_fk = ? AND %s.type = ?", licenseStatusFk, StatusActiveInt).Find(&result).Error
}

// CheckDeviceStatus gets the current status of a device
// if the device has not been recorded in the 'event' table, typeString is empty.
//
func (i transactionEventStore) CheckDeviceStatus(licenseStatusFk int64, deviceId string) (Status, error) {
	var result TransactionEvent
	//`SELECT type FROM event WHERE license_status_fk = ? AND device_id = ? ORDER BY timestamp DESC LIMIT 1`
	return result.Type, i.db.Model(&TransactionEvent{}).Where("license_status_fk = ? AND device_id = ?", licenseStatusFk, deviceId).Find(&result).Error
}
