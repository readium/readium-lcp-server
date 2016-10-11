package licensestatuses

import (
	"time"

	"github.com/readium/readium-lcp-server/transactions"
)

type Updated struct {
	License *time.Time `json:"license,omitempty"`
	Status  *time.Time `json:"status,omitempty"`
}

type Link struct {
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Templated bool   `json:"templated,omitempty" "default false"`
}

type PotentialRights struct {
	End *time.Time `json:"end,omitempty"`
}

type LicenseStatus struct {
	Id              int                  `json:"-"`
	LicenseRef      string               `json:"id"`
	Status          string               `json:"status"`
	Updated         *Updated             `json:"updated,omitempty"`
	Message         string               `json:"message"`
	Links           map[string][]Link    `json:"links,omitempty"`
	DeviceCount     *int                 `json:"device_count,omitempty"`
	PotentialRights *PotentialRights     `json:"potential_rights,omitempty"`
	Events          []transactions.Event `json:"events,omitempty"`
}
