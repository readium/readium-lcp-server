// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/sign"
)

type Key struct {
	Algorithm string `json:"algorithm,omitempty"`
}

type ContentKey struct {
	Key
	Value []byte `json:"encrypted_value,omitempty"`
}

type UserKey struct {
	Key
	Hint     string `json:"text_hint,omitempty"`
	Check    []byte `json:"key_check,omitempty"`
	Value    []byte `json:"value,omitempty"`     //Used for license generation
	HexValue string `json:"hex_value,omitempty"` //Used for license generation
}

type Encryption struct {
	Profile    string     `json:"profile,omitempty"`
	ContentKey ContentKey `json:"content_key"`
	UserKey    UserKey    `json:"user_key"`
}

type Link struct {
	Rel       string `json:"rel"`
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Templated bool   `json:"templated,omitempty" "default false"`
	Size      int64  `json:"length,omitempty"`
	//Digest    []byte `json:"hash,omitempty"`
	Checksum string `json:"hash,omitempty"`
}

type UserInfo struct {
	Id        string   `json:"id"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

type UserRights struct {
	Print *int32     `json:"print,omitempty"`
	Copy  *int32     `json:"copy,omitempty"`
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

const BASIC_PROFILE = "http://readium.org/lcp/basic-profile"
const V1_PROFILE = "http://readium.org/lcp/profile-1.0"

var DefaultLinks map[string]string

type License struct {
	Provider   string          `json:"provider"`
	Id         string          `json:"id"`
	Issued     time.Time       `json:"issued"`
	Updated    *time.Time      `json:"updated,omitempty"`
	Encryption Encryption      `json:"encryption"`
	Links      []Link          `json:"links,omitempty"`
	User       UserInfo        `json:"user"`
	Rights     *UserRights     `json:"rights,omitempty"`
	Signature  *sign.Signature `json:"signature,omitempty"`
	ContentId  string          `json:"-"`
}

type LicenseReport struct {
	Provider  string      `json:"provider"`
	Id        string      `json:"id"`
	Issued    time.Time   `json:"issued"`
	Updated   *time.Time  `json:"updated,omitempty"`
	User      UserInfo    `json:"user,omitempty"`
	Rights    *UserRights `json:"rights"`
	ContentId string      `json:"-"`
}

// source: http://play.golang.org/p/4FkNSiUDMg
// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// Initialize sets a license id and issued date, contentID,
//
func Initialize(contentID string, l *License) {
	// random license id
	uuid, _ := newUUID()
	l.Id = uuid
	// issued datetime is now
	l.Issued = time.Now().UTC().Truncate(time.Second)
	// set the content id
	l.ContentId = contentID
}

// SetLicenseProfile sets the license profile from config
//
func SetLicenseProfile(l *License) {
	// possible profiles are basic and 1.0
	if config.Config.Profile == "1.0" {
		l.Encryption.Profile = V1_PROFILE
	} else {
		l.Encryption.Profile = BASIC_PROFILE
	}
}

// CreateDefaultLinks inits the global var DefaultLinks from config data
// ... DefaultLinks used in several places.
//
func CreateDefaultLinks() {
	configLinks := config.Config.License.Links

	DefaultLinks = make(map[string]string)

	for key := range configLinks {
		DefaultLinks[key] = configLinks[key]
	}
}

// SetDefaultLinks sets a Link array from config links
//
func SetDefaultLinks() []Link {
	links := new([]Link)
	for key := range DefaultLinks {
		link := Link{Href: DefaultLinks[key], Rel: key}
		*links = append(*links, link)
	}
	return *links
}

// SetLicenseLinks sets publication and status links
// l.ContentId must have been set before the call
//
func SetLicenseLinks(l *License, c index.Content) error {

	// set the links
	l.Links = SetDefaultLinks()

	for i := 0; i < len(l.Links); i++ {
		// publication link
		if l.Links[i].Rel == "publication" {
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{publication_id}", l.ContentId, 1)
			l.Links[i].Type = epub.ContentType_EPUB
			l.Links[i].Size = c.Length
			l.Links[i].Title = c.Location
			l.Links[i].Checksum = c.Sha256
		}
		// status link
		if l.Links[i].Rel == "status" {
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{license_id}", l.Id, 1)
			l.Links[i].Type = api.ContentType_LSD_JSON
		}
	}

	return nil
}

// EncryptLicenseFields sets the content key, encrypted user info and key check
//
func EncryptLicenseFields(l *License, c index.Content) error {

	// generate the user key
	encryptionKey := GenerateUserKey(l.Encryption.UserKey)

	// empty the passphrase hash to avoid sending it back to the user
	l.Encryption.UserKey.Value = nil
	l.Encryption.UserKey.HexValue = ""

	// encrypt the content key with the user key
	encrypterContentKey := crypto.NewAESEncrypter_CONTENT_KEY()
	l.Encryption.ContentKey.Algorithm = encrypterContentKey.Signature()
	l.Encryption.ContentKey.Value = encryptKey(encrypterContentKey, c.EncryptionKey, encryptionKey[:])

	// encrypt the user info fields
	encrypterFields := crypto.NewAESEncrypter_FIELDS()
	err := encryptFields(encrypterFields, l, encryptionKey[:])
	if err != nil {
		return err
	}

	// build the key check
	encrypterUserKeyCheck := crypto.NewAESEncrypter_USER_KEY_CHECK()
	l.Encryption.UserKey.Check, err = buildKeyCheck(l.Id, encrypterUserKeyCheck, encryptionKey[:])
	if err != nil {
		return err
	}
	return nil
}

func encryptKey(encrypter crypto.Encrypter, key []byte, kek []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(key)
	encrypter.Encrypt(kek[:], in, &out)
	return out.Bytes()
}

func encryptFields(encrypter crypto.Encrypter, l *License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(&l.User, toEncrypt)

		if !field.IsValid() {
			return fmt.Errorf("The field '%s' is not valid for encrypted. The valid fields are email and name.", toEncrypt)
		}

		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

func getField(u *UserInfo, field string) reflect.Value {
	v := reflect.ValueOf(u).Elem()
	return v.FieldByName(strings.Title(field))
}

// buildKeyCheck
// encrypt the license id with the key used for encrypting content
//
func buildKeyCheck(licenseID string, encrypter crypto.Encrypter, key []byte) ([]byte, error) {
	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(licenseID), &out)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// SignLicense signs a license using the server certificate
//
func SignLicense(l *License, cert *tls.Certificate) error {
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
