// Copyright 2021 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilsd

import (
	"log"
	"testing"

	"github.com/readium/readium-lcp-server/config"
)

// enter here an existing license id
var LicenseID string = "812bbfe8-9a57-4b14-b8f3-4e0fc6e841c0"

func TestGetUserData(t *testing.T) {

	// enter here a valid URL
	config.Config.LsdServer.UserDataUrl = "http://xx.xx.xx.xx:9999/aaaaa/{license_id}/aaaa"
	// enter here valid credentials
	config.Config.CMSAccessAuth.Username = "xxxxx"
	config.Config.CMSAccessAuth.Password = "xxxxx"

	log.Println("username ", config.Config.CMSAccessAuth.Username)

	userData, err := getUserData(LicenseID)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	if userData.Name == "" {
		t.Error("Unexpected user name")
	}
}

func TestInitPartialLicense(t *testing.T) {

	userData := UserData{
		ID:             "123-123-123",
		Name:           "John Doe",
		Email:          "jdoe@example.com",
		Hint:           "Good hint",
		PassphraseHash: "faeb00ca518bea7cb11a7ef31f63183b489b1b6eadb792bec64a03b3f6ff80a8",
	}

	plic, err := initPartialLicense(LicenseID, userData)

	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	if plic.User.ID != userData.ID {
		t.Error("Unexpected user ID")
	}

	if plic.Encryption.UserKey.Algorithm != Sha256_URL {
		t.Error("Unexpected UserKey algorithm")
	}
}

func TestFetchLicense(t *testing.T) {

	// enter here a valid URL
	config.Config.LcpServer.PublicBaseUrl = "http://xx.xx.xx.xx:9999"
	// enter here valid credentials
	config.Config.LcpUpdateAuth.Username = "xxxx"
	config.Config.LcpUpdateAuth.Password = "xxxx"

	userData := UserData{
		ID:             "123-123-123",
		Name:           "John Doe",
		Email:          "jdoe@example.com",
		Hint:           "Good hint",
		PassphraseHash: "faeb00ca518bea7cb11sdf434b6183b489b1b6eadb792bec64a03b3f6ff80a8",
	}

	plic, _ := initPartialLicense(LicenseID, userData)

	_, err := fetchLicense(plic)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
}
