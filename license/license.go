package license

import (
	"crypto/rand"
	"fmt"

	"github.com/jpbougie/lcpserve/sign"

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
	Hint string `json:"hint"`
}

type Encryption struct {
	Profile    string     `json:"profile"`
	ContentKey ContentKey `json:"content_key"`
	UserKey    UserKey    `json:"user_key"`
}

type Link struct {
	Href   string `json:"href"`
	Type   string `json:"type,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Digest []byte `json:"digest,omitempty"`
}

type UserInfo struct {
	Provider  string   `json:"provider"`
	Id        string   `json:"id"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

type UserRights struct {
	Print    int32     `json:"print"`
	Copy     int32     `json:"copy"`
	TTS      bool      `json:"tts"`
	Editable bool      `json:"edit"`
	Start    time.Time `json:"start,omitempty"`
	End      time.Time `json:"end,omitempty"`
}

const DEFAULT_PROFILE = "http://readium.org/lcp/profile-1.0"

type License struct {
	Id         string          `json:"id"`
	Date       time.Time       `json:"date"`
	Encryption Encryption      `json:"encryption"`
	Links      map[string]Link `json:"links"`
	User       UserInfo        `json:"user"`
	Rights     *UserRights     `json:"rights,omitempty"`
	Signature  *sign.Signature `json:"signature,omitempty"`
}

func New() License {
	l := License{Links: map[string]Link{}}
	return Prepare(l)
}

func Prepare(l License) License {
	uuid, _ := newUUID()
	l.Id = uuid
	l.Date = time.Now()

	l.Encryption.Profile = DEFAULT_PROFILE

	return l
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
