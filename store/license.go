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

package store

import (
	"fmt"
	"github.com/readium/readium-lcp-server/sign"
	"github.com/satori/go.uuid"
	"time"
)

type (
	Key struct {
		Algorithm string `json:"algorithm,omitempty"`
	}

	LicenseContentKey struct {
		Key
		Value []byte `json:"encrypted_value,omitempty"`
	}

	LicenseUserKey struct {
		Key
		Hint  string `json:"text_hint,omitempty"`
		Check string `json:"key_check,omitempty"`
		Value string `json:"value,omitempty"` // Used for the license request
	}

	LicenseEncryption struct {
		Profile    string            `json:"profile,omitempty"`
		ContentKey LicenseContentKey `json:"content_key"`
		UserKey    LicenseUserKey    `json:"user_key"`
	}

	LicenseUserRights struct {
		Print *NullInt  `json:"print,omitempty"`
		Copy  *NullInt  `json:"copy,omitempty"`
		Start *NullTime `json:"start,omitempty"`
		End   *NullTime `json:"end,omitempty"`
	}

	LicenseLinksCollection []*LicenseLink
	LicenseLink            struct {
		Rel       string `json:"rel"`
		Href      string `json:"href"`
		Type      string `json:"type,omitempty"`
		Title     string `json:"title,omitempty"`
		Profile   string `json:"profile,omitempty"`
		Templated bool   `json:"templated,omitempty"`
		Size      int64  `json:"length,omitempty"`
		Checksum  string `json:"hash,omitempty"`
		//Digest    []byte `json:"hash,omitempty"`
	}

	LicensesCollection []*License
	License            struct {
		Id         string                 `json:"id" sql:"NOT NULL" gorm:"primary_key"`
		UserId     string                 `json:"-" sql:"NOT NULL"`
		Provider   string                 `json:"provider" sql:"NOT NULL"`
		Issued     time.Time              `json:"issued" sql:"DEFAULT:current_timestamp;NOT NULL"`
		Updated    *NullTime              `json:"updated,omitempty" sql:"DEFAULT NULL"`
		Print      *NullInt               `json:"-" sql:"DEFAULT NULL" gorm:"column:rights_print"`
		Copy       *NullInt               `json:"-" sql:"DEFAULT NULL" gorm:"column:rights_copy"`
		Start      *NullTime              `json:"-" sql:"DEFAULT NULL" gorm:"column:rights_start"`
		End        *NullTime              `json:"-" sql:"DEFAULT NULL" gorm:"column:rights_end"`
		Rights     *LicenseUserRights     `json:"rights,omitempty" gorm:"-"`
		ContentId  string                 `json:"contentId" gorm:"column:content_fk" sql:"NOT NULL"`
		LSDStatus  int32                  `json:"-"` // TODO : never used. is this work in progress?
		User       *User                  `json:"user,omitempty" gorm:"-"`
		Content    *Content               `json:"-" gorm:"associationForeignKey:Id"`
		Encryption LicenseEncryption      `json:"encryption,omitempty"`
		Links      LicenseLinksCollection `json:"links,omitempty"`
		Signature  *sign.Signature        `json:"signature,omitempty"`
	}
)

// Implementation of GORM Tabler
func (l *License) TableName() string {
	return LCPLicenseTableName
}

// restores fields for json response
// Implementation of GORM callback
func (l *License) AfterFind() error {
	// nullify updated to be omitted if empty
	if l.Updated != nil && !l.Updated.Valid {
		l.Updated = nil
	}

	// transfer from db to json
	if l.Print != nil {
		if l.Rights == nil {
			l.Rights = &LicenseUserRights{}
		}
		l.Rights.Print = l.Print
	}
	if l.Copy != nil {
		if l.Rights == nil {
			l.Rights = &LicenseUserRights{}
		}
		l.Rights.Copy = l.Copy
	}
	if l.Start != nil {
		if l.Rights == nil {
			l.Rights = &LicenseUserRights{}
		}
		l.Rights.Start = l.Start
	}
	if l.End != nil {
		if l.Rights == nil {
			l.Rights = &LicenseUserRights{}
		}
		l.Rights.End = l.End
	}
	return nil
}

// restore infos from db
// Implementation of GORM callback
func (l *License) BeforeSave() error {
	// Create uuid
	if l.Id == "" {
		uid, errU := uuid.NewV4()
		if errU != nil {
			return errU
		}
		l.Id = uid.String()
	}
	if l.User != nil {
		if err := l.User.BeforeSave(); err != nil {
			return err
		}
	}
	// transfer from json to db
	if l.Rights != nil {
		l.Copy = l.Rights.Copy
		l.Print = l.Rights.Print
		l.Start = l.Rights.Start
		l.End = l.Rights.End
	}
	return nil
}

// get license, check mandatory information in the input body
//
func (l *License) CheckGetLicenseInput() error {
	if l.Encryption.UserKey.Hint == "" {
		return fmt.Errorf("Mandatory info missing in the input body : User hint is missing.")
	}
	if l.Encryption.UserKey.Value == "" {
		return fmt.Errorf("Mandatory info missing in the input body : hashed passphrase is missing.")
	}
	if l.Encryption.UserKey.Algorithm == "" {
		//log.Println("User passphrase hash algorithm is missing, set default value")
		// the only valid value in LCP basic and 10 profiles is sha256
		l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	}
	return nil
}

// generate license, check mandatory information in the input body
//
func (l *License) CheckGenerateLicenseInput() error {
	if l.Provider == "" {
		return fmt.Errorf("Mandatory info missing in the input body  : license provider is missing.")
	}
	if l.User.UUID == "" {
		return fmt.Errorf("Mandatory info missing in the input body : user identificator is missing.")
	}
	// check userkey hint, value and algorithm
	err := l.CheckGetLicenseInput()
	return err
}

// get license, copy useful data from licIn to LicOut
//
func (l *License) CopyInputToLicense(licOut *License) {
	// copy the hashed passphrase, user hint and algorithm
	licOut.Encryption.UserKey = l.Encryption.UserKey
	// copy optional user information
	licOut.User.Email = l.User.Email
	licOut.User.Name = l.User.Name
	licOut.User.Encrypted = l.User.Encrypted
}

// normalize the start and end date, UTC, no milliseconds
//
func (l *License) SetRights() {
	// if Start nor End valid, we have nothing to save ??
	if !l.Rights.Start.Valid || !l.Rights.End.Valid {
		panic("SetRights not valid. Should be valid ?")
	}
	if l.Rights.Start.Valid {
		// normalize the start and end date, UTC, no milliseconds
		l.Rights.Start.Time = l.Rights.Start.Time.UTC().Truncate(time.Second)
	}
	if l.Rights.End.Valid {
		// normalize the start and end date, UTC, no milliseconds
		l.Rights.End.Time = l.Rights.End.Time.UTC().Truncate(time.Second)
	}
}

// Initialize sets a license id and issued date, contentID,
//
func (l *License) Initialize(contentID string) error {
	// random license id
	uid, errU := uuid.NewV4()
	if errU != nil {
		return errU
	}
	l.Id = uid.String()
	// issued datetime is now
	l.Issued = time.Now().UTC().Truncate(time.Second)
	// set the content id
	l.ContentId = contentID
	return nil
}

// UpdateRights
//
func (s *licenseStore) UpdateRights(l *License) error {
	var result License
	err := s.db.Where(License{Id: l.Id}).Find(&result).Error
	if err != nil {
		return err
	}
	return s.db.Model(&result).Updates(map[string]interface{}{
		"rights_print": l.Rights.Print,
		"rights_copy":  l.Rights.Copy,
		"rights_start": l.Rights.Start.Time,
		"rights_end":   l.Rights.End.Time,
		"updated":      Now().Time,
	}).Error
}

// Add creates a new record in the license table
//
func (s *licenseStore) Add(l *License) error {
	return s.db.Debug().Create(l).Error
}

// Update updates a record in the license table
//
func (s *licenseStore) Update(l *License) error {
	l.Updated = Now()
	return s.db.Save(l).Error
}

// UpdateLsdStatus
//
func (s *licenseStore) UpdateLsdStatus(id string, status int32) error {
	var result License
	err := s.db.Where(License{Id: id}).Find(&result).Error
	if err != nil {
		return err
	}
	return s.db.Model(&result).Updates(map[string]interface{}{"lsd_status": status}).Error
}

// Get a license from the db
//
func (s *licenseStore) Get(id string) (*License, error) {
	var result License
	return &result, s.db.Where(License{Id: id}).Find(&result).Error
}

// ListAll lists all licenses in ante-chronological order
// pageNum starts at 0
//
func (s *licenseStore) ListAll(page int, pageNum int) (LicensesCollection, error) {
	var result LicensesCollection
	return result, s.db.Offset(pageNum * page).Limit(page).Order("issued DESC").Find(&result).Error
}

// List lists licenses for a given ContentId
// pageNum starting at 0
//
func (s *licenseStore) List(contentID string, page int, pageNum int) (LicensesCollection, error) {
	var result LicensesCollection
	return result, s.db.Where("content_fk = ?", contentID).Offset(pageNum * page).Limit(page).Order("issued DESC").Find(&result).Error
}
