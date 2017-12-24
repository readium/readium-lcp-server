// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/sign"
)

type Key struct {
	Algorithm string `json:"algorithm"`
}

type ContentKey struct {
	Key
	Value []byte `json:"encrypted_value"`
}

type UserKey struct {
	Key
	Hint  string `json:"text_hint"`
	Check []byte `json:"key_check,omitempty"`
	Value []byte `json:"value,omitempty"` //Used for the license request
}

type Encryption struct {
	Profile    string     `json:"profile"`
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
	Links      []Link          `json:"links"`
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
	Rights    *UserRights `json:"rights,omitempty"`
	ContentId string      `json:"-"`
}

func CreateLinks() {
	var configLinks map[string]string = config.Config.License.Links

	DefaultLinks = make(map[string]string)

	for key := range configLinks {
		DefaultLinks[key] = configLinks[key]
	}
}

func DefaultLinksCopy() []Link {
	links := new([]Link)
	for key := range DefaultLinks {
		link := Link{Href: DefaultLinks[key], Rel: key}
		*links = append(*links, link)
	}
	return *links
}

func New() License {
	l := License{}
	Prepare(&l)
	return l
}

func Prepare(l *License) {
	uuid, _ := newUUID()
	l.Id = uuid

	l.Issued = time.Now().UTC().Truncate(time.Second)

	if l.Links == nil {
		l.Links = DefaultLinksCopy()
	}

	if l.Rights == nil {
		l.Rights = new(UserRights)
	}

	if config.Config.Profile == "1.0" {
		l.Encryption.Profile = V1_PROFILE
	} else {
		l.Encryption.Profile = BASIC_PROFILE
	}
}

func createForeigns(l *License) {
	l.Encryption = Encryption{}
	l.Encryption.UserKey = UserKey{}
	l.User = UserInfo{}
	l.Rights = new(UserRights)
	l.Signature = new(sign.Signature)

	l.Links = DefaultLinksCopy()

	if config.Config.Profile == "1.0" {
		l.Encryption.Profile = V1_PROFILE
	} else {
		l.Encryption.Profile = BASIC_PROFILE
	}
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
