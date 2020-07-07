// Copyright 2020 Readium Foundation. All rights reserved.
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
	ID        string   `json:"id"`
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

var DefaultLinks map[string]string

type License struct {
	Provider   string          `json:"provider"`
	ID         string          `json:"id"`
	Issued     time.Time       `json:"issued"`
	Updated    *time.Time      `json:"updated,omitempty"`
	Encryption Encryption      `json:"encryption"`
	Links      []Link          `json:"links,omitempty"`
	User       UserInfo        `json:"user"`
	Rights     *UserRights     `json:"rights,omitempty"`
	Signature  *sign.Signature `json:"signature,omitempty"`
	ContentID  string          `json:"-"`
}

type LicenseReport struct {
	Provider  string      `json:"provider"`
	ID        string      `json:"id"`
	Issued    time.Time   `json:"issued"`
	Updated   *time.Time  `json:"updated,omitempty"`
	User      UserInfo    `json:"user,omitempty"`
	Rights    *UserRights `json:"rights"`
	ContentID string      `json:"-"`
}

// EncryptionProfile is an enum of possible encryption profiles
type EncryptionProfile int

// Declare typed constants for Encryption Profile
const (
	BasicProfile EncryptionProfile = iota
	V1Profile
)

func (profile EncryptionProfile) String() string {

	var profileURL string
	switch profile {
	case BasicProfile:
		profileURL = "http://readium.org/lcp/basic-profile"
	case V1Profile:
		profileURL = "http://readium.org/lcp/profile-1.0"
	default:
		profileURL = "unknown-profile"
	}

	return profileURL
}

// SetLicenseProfile sets the license profile from config
func SetLicenseProfile(l *License) {

	// possible profiles are basic and 1.0
	var ep EncryptionProfile
	if config.Config.Profile == "1.0" {
		ep = V1Profile
	} else {
		ep = BasicProfile
	}
	l.Encryption.Profile = ep.String()
}

// newUUID generates a random UUID according to RFC 4122
// source: http://play.golang.org/p/4FkNSiUDMg
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
func Initialize(contentID string, l *License) {

	// random license id
	uuid, _ := newUUID()
	l.ID = uuid
	// issued datetime is now
	l.Issued = time.Now().UTC().Truncate(time.Second)
	// set the content id
	l.ContentID = contentID
}

// CreateDefaultLinks inits the global var DefaultLinks from config data
// ... DefaultLinks used in several places.
func CreateDefaultLinks() {

	configLinks := config.Config.License.Links

	DefaultLinks = make(map[string]string)

	for key := range configLinks {
		DefaultLinks[key] = configLinks[key]
	}
}

// SetDefaultLinks sets a Link array from config links
func SetDefaultLinks() []Link {

	links := new([]Link)
	for key := range DefaultLinks {
		link := Link{Href: DefaultLinks[key], Rel: key}
		*links = append(*links, link)
	}
	return *links
}

// SetLicenseLinks sets publication and status links
// l.ContentID must have been set before the call
func SetLicenseLinks(l *License, c index.Content) error {

	// set the links
	l.Links = SetDefaultLinks()

	for i := 0; i < len(l.Links); i++ {
		// publication link
		if l.Links[i].Rel == "publication" {
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{publication_id}", l.ContentID, 1)
			l.Links[i].Type = c.Type
			l.Links[i].Size = c.Length
			l.Links[i].Title = c.Location
			l.Links[i].Checksum = c.Sha256
		}
		// status link
		if l.Links[i].Rel == "status" {
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{license_id}", l.ID, 1)
			l.Links[i].Type = api.ContentType_LSD_JSON
		}
	}

	return nil
}

// EncryptLicenseFields sets the content key, encrypted user info and key check
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
	l.Encryption.UserKey.Check, err = buildKeyCheck(l.ID, encrypterUserKeyCheck, encryptionKey[:])
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
func buildKeyCheck(licenseID string, encrypter crypto.Encrypter, key []byte) ([]byte, error) {

	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(licenseID), &out)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// SignLicense signs a license using the server certificate
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
