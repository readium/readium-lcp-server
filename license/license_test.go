// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"testing"

	"github.com/readium/readium-lcp-server/config"
)

func TestLicense(t *testing.T) {
	config.Config.Profile = "1.0"

	l := License{}
	contentID := "1234-1234-1234-1234"
	Initialize(contentID, &l)
	if l.ID == "" {
		t.Error("Should have an id")
	}

	err := SetLicenseProfile(&l)
	if err != nil {
		t.Error(err.Error())
	}
}
