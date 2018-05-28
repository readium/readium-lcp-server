package model_test

import (
	"database/sql"
	"github.com/dmgk/faker"
	"github.com/readium/readium-lcp-server/model"
	"testing"
)

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
			entity.LicenseUpdated = model.TruncatedNow()
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
	updatedLicense.LicenseUpdated = model.TruncatedNow()
	updatedLicense.Status = model.StatusExpired
	updatedLicense.StatusUpdated = model.TruncatedNow()
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
