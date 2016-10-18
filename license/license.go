// Copyright (c) 2016 Readium Founation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. 

package license

import (
	"crypto/rand"
	"fmt"

	"github.com/readium/readium-lcp-server/sign"

	"io"
	"time"
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
	Hint       string `json:"text_hint"`
	Check      []byte `json:"key_check,omitempty"`
	Value      []byte `json:"value,omitempty"`       //Used for the license request
	ClearValue string `json:"clear_value,omitempty"` //Used for the license request
}

type Encryption struct {
	Profile    string     `json:"profile"`
	ContentKey ContentKey `json:"content_key"`
	UserKey    UserKey    `json:"user_key"`
}

type Link struct {
	Href   string `json:"href"`
	Type   string `json:"type,omitempty"`
	Size   int64  `json:"length,omitempty"`
	Digest []byte `json:"hash,omitempty"`
}

type UserInfo struct {
	Id        string   `json:"id"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

type UserRights struct {
	Print    *int32     `json:"print,omitempty"`
	Copy     *int32     `json:"copy,omitempty"`
	TTS      bool       `json:"tts"`
	Editable bool       `json:"edit"`
	Start    *time.Time `json:"start,omitempty"`
	End      *time.Time `json:"end,omitempty"`
}

var DefaultRights = UserRights{
	TTS:      true,
	Editable: false,
}

const DEFAULT_PROFILE = "http://readium.org/lcp/profile-1.0"

type License struct {
	Provider   string          `json:"provider"`
	Id         string          `json:"id"`
	Issued     time.Time       `json:"issued"`
	Updated    time.Time       `json:"updated"`
	Encryption Encryption      `json:"encryption"`
	Links      map[string]Link `json:"links"`
	User       UserInfo        `json:"user"`
	Rights     *UserRights     `json:"rights,omitempty"`
	Signature  *sign.Signature `json:"signature,omitempty"`
}

func New() License {
	l := License{Links: map[string]Link{}}
	Prepare(&l)
	return l
}

func Prepare(l *License) {
	uuid, _ := newUUID()
	l.Id = uuid
	l.Issued = time.Now()
	l.Updated = l.Issued

	if l.Links == nil {
		l.Links = map[string]Link{}
	}

	if l.Rights == nil {
		l.Rights = &DefaultRights
	}

	l.Encryption.Profile = DEFAULT_PROFILE
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
