/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package model_test

import (
	"database/sql"
	"github.com/dmgk/faker"
	"github.com/readium/readium-lcp-server/model"
	"testing"
	"time"
)

func TestLicenseStatusStore_DeleteByLicenseIds(t *testing.T) {
	stor.LicenseStatus().DeleteByLicenseIds("ed55a97b-f100-4388-a49e-35e69643b5cd,f61a9829-6770-4bdf-b040-3a235e63699c,32638118-5a55-4fe9-b2e9-51c5b28a9986,55e3a32f-5ac5-43d4-9e88-18a5aaaba5cf,45f3edcb-fc7a-4969-90ea-98be16d3070b,e49bd52f-8cbc-4be0-9c1f-b5addfde0a4f,b346dc6a-72a3-4856-b6c4-26999ed10573")
}

func TestLicenseStatusStore_Add(t *testing.T) {
	for i := 0; i < 30; i++ {
		// Create uuid
		uid, errU := model.NewUUID()
		if errU != nil {
			t.Fatalf("Error creating UUID: %v", errU)
		}
		entity := &model.LicenseStatus{
			LicenseRef:  uid.String(),
			Status:      model.StatusActive,
			DeviceCount: &model.NullInt{NullInt64: sql.NullInt64{Int64: faker.RandomInt64(0, 65535), Valid: true}},
		}

		v1 := faker.RandomInt64(0, 2)

		if v1 == 1 {
			entity.LicenseUpdated = time.Now()
		}

		if v1 == 0 {
			entity.PotentialRightsEnd = model.TruncatedNow()
		}

		err := stor.LicenseStatus().Add(entity)
		if err != nil {
			t.Fatalf("Error creating license status : %v", err)
		}
	}
}

func TestLicenseStatusStore_List(t *testing.T) {
	licenses, err := stor.LicenseStatus().List(1, 50, 0)
	if err != nil {
		t.Fatalf("Error reading : %v", err)
	}
	for _, license := range licenses {
		t.Logf("%#v", license)
	}
}

func TestLicenseStatusStore_Update(t *testing.T) {
	licenses, err := stor.LicenseStatus().List(1, 50, 0)
	if err != nil {
		t.Fatalf("Error reading : %v", err)
	}
	updatedLicense := licenses[0]
	updatedLicense.LicenseUpdated = time.Now()
	updatedLicense.Status = model.StatusExpired
	updatedLicense.StatusUpdated = time.Now()
	updatedLicense.LicenseRef = "Bad license ref"
	err = stor.LicenseStatus().Update(updatedLicense)
	if err != nil {
		t.Fatalf("Error updating : %v", err)
	}
	afterUpdate, err := stor.LicenseStatus().GetByLicenseId("Bad license ref")
	if err != nil {
		t.Fatalf("Error finding : %v", err)
	}
	t.Logf("%#v", afterUpdate)
}

func TestLicenseStatusStore_GetByLicenseId(t *testing.T) {
	licenseStatus, err := stor.LicenseStatus().GetByLicenseId("Bad license ref")
	if err != nil {
		t.Fatalf("Error finding : %v", err)
	}
	t.Logf("%#v", licenseStatus)
}
