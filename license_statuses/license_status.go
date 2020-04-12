// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package licensestatuses

import (
	"time"

	"github.com/readium/readium-lcp-server/transactions"
)

// Updated represents license and status document timestamps
type Updated struct {
	License *time.Time `json:"license,omitempty"`
	Status  *time.Time `json:"status,omitempty"`
}

// Link represents a link object
type Link struct {
	Rel       string `json:"rel"`
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Templated bool   `json:"templated,omitempty" "default false"`
}

// PotentialRights represents the maximal extension time of a loan
type PotentialRights struct {
	End *time.Time `json:"end,omitempty"`
}

// LicenseStatus represents a license status
type LicenseStatus struct {
	ID                int                  `json:"-"`
	LicenseRef        string               `json:"id"`
	Status            string               `json:"status"`
	Updated           *Updated             `json:"updated,omitempty"`
	Message           string               `json:"message"`
	Links             []Link               `json:"links,omitempty"`
	DeviceCount       *int                 `json:"device_count,omitempty"`
	PotentialRights   *PotentialRights     `json:"potential_rights,omitempty"`
	Events            []transactions.Event `json:"events,omitempty"`
	CurrentEndLicense *time.Time           `json:"-"`
}
