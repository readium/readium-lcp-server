package history

import (
	"strconv"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/transactions"
)

const (
	STATUS_READY     = "ready"
	STATUS_ACTIVE    = "active"
	STATUS_REVOKED   = "revoked"
	STATUS_RETURNED  = "returned"
	STATUS_CANCELLED = "cancelled"
	STATUS_EXPIRED   = "expired"
)

var statuses = map[int]string{
	0: STATUS_READY,
	1: STATUS_ACTIVE,
	2: STATUS_REVOKED,
	3: STATUS_RETURNED,
	4: STATUS_CANCELLED,
	5: STATUS_EXPIRED,
}

type Updated struct {
	License *time.Time `json:"license,omitempty"`
	Status  *time.Time `json:"status,omitempty"`
}

type Link struct {
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Templated bool   `json:"templated,omitempty" "default false"`
	Profile   string `json:"profile,omitempty"`
}

type PotentialRights struct {
	End time.Time `json:"end,omitempty"`
}

type LicenseStatus struct {
	Id              int                  `json:"-"`
	LicenseRef      string               `json:"id"`
	Status          string               `json:"status"`
	Updated         *Updated             `json:"updated,omitempty"`
	Message         string               `json:"message"`
	Links           map[string][]Link    `json:"links"`
	DeviceCount     int                  `json:"device_count"`
	PotentialRights *PotentialRights     `json:"potential_rights,omitempty"`
	Events          []transactions.Event `json:"events,omitempty"`
}

func getStatus(statusDB int64, status *string) {
	resultStr := reverse(strconv.FormatInt(statusDB, 2))

	if count := strings.Count(resultStr, "1"); count == 1 {
		index := strings.Index(resultStr, "1")

		if len(statuses) >= index+1 {
			*status = statuses[index]
		}
	}
}

func setStatus(status string) (int64, error) {
	reg := make([]string, len(statuses))

	for key := range statuses {
		if statuses[key] == status {
			reg[key] = "1"
		} else {
			reg[key] = "0"
		}
	}

	resultStr := reverse(strings.Join(reg[:], ""))

	statusDB, err := strconv.ParseInt(resultStr, 2, 64)
	return statusDB, err
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
