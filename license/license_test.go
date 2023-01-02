// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"log"
	"regexp"
	"testing"

	"github.com/readium/readium-lcp-server/config"
)

func TestLicense(t *testing.T) {
	l := License{}
	contentID := "1234-1234-1234-1234"
	Initialize(contentID, &l)
	if l.ID == "" {
		t.Error("Should have an id")
	}

	config.Config.Profile = "basic"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/basic-profile" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "1.0"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-1.0" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.0"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.0" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.1"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.1" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.2"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.2" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.3"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.3" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.4"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.4" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.5"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.5" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.6"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.6" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.7"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.7" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.8"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.8" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.9"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if l.Encryption.Profile != "http://readium.org/lcp/profile-2.9" {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
	config.Config.Profile = "2.x"
	SetLicenseProfile(&l)
	log.Print(l.Encryption.Profile)

	if match, _ := regexp.MatchString("^http://readium.org/lcp/profile-2.[0-9]$", l.Encryption.Profile); match == false {
		t.Errorf("Expected '1.0', '2.x' or 'basic', got %s", l.Encryption.Profile)
	}
}
