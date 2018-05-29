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
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/readium/readium-lcp-server/lib/crypto"
	"github.com/readium/readium-lcp-server/lib/sign"
	"reflect"
	"strings"
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

var DefaultLinks map[string]string

// SetLinks sets publication and status links
// l.ContentId must have been set before the call
//
func (l *License) SetLinks(c *Content) error {
	// set the links
	l.Links = make(LicenseLinksCollection, 0, 0)
	for key := range DefaultLinks {
		l.Links = append(l.Links, &LicenseLink{Href: DefaultLinks[key], Rel: key})
	}

	for i := 0; i < len(l.Links); i++ {
		switch l.Links[i].Rel {
		// publication link
		case "publication":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{publication_id}", l.ContentId, 1)
			//l.Links[i].Type = "application/epub+zip"
			l.Links[i].Size = c.Length
			l.Links[i].Title = c.Location
			l.Links[i].Checksum = c.Sha256
			// status link
		case "status":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{license_id}", l.Id, 1)
			//l.Links[i].Type = "application/vnd.readium.license.status.v1.0+json"

		}

	}
	return nil
}

// SignLicense signs a license using the server certificate
//
func (l *License) SignLicense(cert *tls.Certificate) error {
	sig, err := sign.NewSigner(cert)
	if err != nil {
		return err
	}
	res, err := sig.Sign(l)
	if err != nil {
		return err
	}
	l.Signature = &res

	return nil
}

// EncryptLicenseFields sets the content key, encrypted user info and key check
//
func (l *License) EncryptLicenseFields(content *Content) error {

	// generate the user key
	encryptionKey := []byte(l.Encryption.UserKey.Value)

	// empty the passphrase hash to avoid sending it back to the user
	l.Encryption.UserKey.Value = ""

	// encrypt the content key with the user key
	encrypterContentKey := crypto.NewAESEncrypterContentKey()
	l.Encryption.ContentKey.Algorithm = encrypterContentKey.Signature()
	l.Encryption.ContentKey.Value = encryptKey(encrypterContentKey, content.EncryptionKey, encryptionKey[:])

	// encrypt the user info fields
	encrypterFields := crypto.NewAESEncrypterFields()
	err := l.encryptFields(encrypterFields, encryptionKey[:])
	if err != nil {
		return err
	}

	// build the key check
	encrypterUserKeyCheck := crypto.NewAESEncrypterUserKeyCheck()
	chk, err := buildKeyCheck(l.Id, encrypterUserKeyCheck, encryptionKey[:])
	if err != nil {
		return err
	}
	l.Encryption.UserKey.Check = string(chk)
	return nil
}

func (l *License) encryptFields(encrypter crypto.Encrypter, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := l.User.getField(toEncrypt)
		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

// Implementation of Stringer
func (c LicensesCollection) GoString() string {
	result := ""
	for _, e := range c {
		result += e.GoString() + "\n"
	}
	return result
}

func (l License) Validate() error {
	if l.ContentId == "" {
		return errors.New("content id is invalid")
	}
	return nil
}

// Implementation of Stringer
func (l License) GoString() string {
	return "ID : " + l.Id + " user ID : " + l.UserId
}

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
		uid, errU := NewUUID()
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
func (l *License) ValidateEncryption() error {
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
func (l *License) ValidateProviderAndUser() error {
	if l.Provider == "" {
		return fmt.Errorf("Mandatory info missing in the input body  : license provider is missing.")
	}
	if l.User == nil || l.User.UUID == "" {
		return fmt.Errorf("Mandatory info missing in the input body : user identificator is missing.")
	}
	// check userkey hint, value and algorithm
	return l.ValidateEncryption()
}

// get license, copy useful data from licIn to LicOut
//
func (l *License) CopyInputToLicense(lic *License) {
	// copy the hashed passphrase, user hint and algorithm
	lic.Encryption.UserKey = l.Encryption.UserKey
	// copy optional user information
	lic.User.Email = l.User.Email
	lic.User.Name = l.User.Name
	lic.User.Encrypted = l.User.Encrypted
}

// Initialize sets a license id and issued date, contentID,
//
func (l *License) Initialize(contentID string) error {
	// TODO : maybe move to validation ?
	// checking rights
	if l.Rights == nil || l.Rights.Start == nil || l.Rights.End == nil || !l.Rights.Start.Valid || !l.Rights.End.Valid {
		return errors.New("rights not valid")
	}
	// random license id
	uid, errU := NewUUID()
	if errU != nil {
		return errU
	}
	l.Id = uid.String()
	// issued datetime is now
	l.Issued = time.Now().UTC().Truncate(time.Second)
	// set the content id
	l.ContentId = contentID
	// normalize the start and end date, UTC, no milliseconds
	if l.Rights.Start.Valid {
		// normalize the start and end date, UTC, no milliseconds
		l.Rights.Start.Time = l.Rights.Start.Time.UTC().Truncate(time.Second)
	}
	if l.Rights.End.Valid {
		// normalize the start and end date, UTC, no milliseconds
		l.Rights.End.Time = l.Rights.End.Time.UTC().Truncate(time.Second)
	}
	return nil
}

func (l *License) Update(lic *License) {
	// update existingLicense using information found in lic
	if lic.User.UUID != "" {
		l.User.UUID = lic.User.UUID
	}
	if lic.Provider != "" {
		l.Provider = lic.Provider
	}
	if lic.ContentId != "" {
		l.ContentId = lic.ContentId
	}
	if lic.Rights.Print.Valid {
		l.Rights.Print = lic.Rights.Print
	}
	if lic.Rights.Copy.Valid {
		l.Rights.Copy = lic.Rights.Copy
	}
	if lic.Rights.Start.Valid {
		l.Rights.Start = lic.Rights.Start
	}
	if lic.Rights.End.Valid {
		l.Rights.End = lic.Rights.End
	}
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
	return s.db.Create(l).Error
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

// Counts licenses for pagination
func (s *licenseStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(&License{}).Count(&count).Error
}

// ListAll lists all licenses in ante-chronological order
// pageNum starts at 0
//
func (s *licenseStore) ListAll(page int64, pageNum int64) (LicensesCollection, error) {
	var result LicensesCollection
	return result, s.db.Offset(pageNum * page).Limit(page).Order("issued DESC").Find(&result).Error
}

// Counts licenses for a given ContentId
func (s *licenseStore) CountForContentId(contentID string) (int64, error) {
	var count int64
	return count, s.db.Model(&License{}).Where("content_fk = ?", contentID).Count(&count).Error
}

// List lists licenses for a given ContentId
// pageNum starting at 0
//
func (s *licenseStore) List(contentID string, page, pageNum int64) (LicensesCollection, error) {
	var result LicensesCollection
	return result, s.db.Where("content_fk = ?", contentID).Offset(pageNum * page).Limit(page).Order("issued DESC").Find(&result).Error
}
