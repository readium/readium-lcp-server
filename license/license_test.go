// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"testing"
)

func TestLicense(t *testing.T) {
	l := License{}
	contentID := "1234-1234-1234-1234"
	Initialize(contentID, &l)
	if l.ID == "" {
		t.Error("Should have an id")
	}

	SetLicenseProfile(&l)

	basic := "http://readium.org/lcp/basic-profile"
	one := "http://readium.org/lcp/profile-1.0"
	if l.Encryption.Profile != one && l.Encryption.Profile != basic {
		t.Errorf("Expected '%s' or '%s', got %s", basic, one, l.Encryption.Profile)
	}
}
