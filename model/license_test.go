package model_test

import (
	"database/sql"
	"encoding/json"
	"github.com/dmgk/faker"
	"github.com/readium/readium-lcp-server/model"
	"testing"
)

func TestLicenseStore_Add(t *testing.T) {
	// reading all
	contents, err := stor.Content().List()
	if err != nil {
		t.Fatalf("Error retrieving contents : %v", err)
	}
	if len(contents) == 0 {
		t.Skipf("You forgot to create contents?")
	}
	// for each content, generate a license
	for _, content := range contents {
		license := &model.License{
			Content:  content,
			Provider: faker.Name().Name(),
			Rights: &model.LicenseUserRights{
				Start: model.TruncatedNow(),
				End:   model.TruncatedNow(),
				Print: &model.NullInt{NullInt64: sql.NullInt64{Int64: faker.RandomInt64(0, 20), Valid: true}},
				Copy:  &model.NullInt{NullInt64: sql.NullInt64{Int64: faker.RandomInt64(0, 300), Valid: true}},
			},
		}
		err = stor.License().Add(license)
		if err != nil {
			t.Fatalf("Error retrieving contents : %v", err)
		}
	}
}

func TestLicenseStore_ListAll(t *testing.T) {
	licenses, err := stor.License().ListAll(50, -1)
	if err != nil {
		t.Fatalf("Error retrieving licenses : %v", err)
	}
	t.Logf("Found %d licenses.", len(licenses))
	for _, license := range licenses {
		t.Logf("%#v", license)
	}
}

func TestLicenseStore_Get(t *testing.T) {
	licenses, err := stor.License().ListAll(50, -1)
	if err != nil {
		t.Fatalf("Error retrieving licenses : %v", err)
	}
	if len(licenses) == 0 {
		t.Skip("You forgot to create some licenses?")
	}
	license, err := stor.License().Get(licenses[0].Id)
	if err != nil {
		t.Fatalf("Error retrieving license with id %q : %v", licenses[0].Id, err)
	}
	jsonPayload, err := json.MarshalIndent(license, " ", " ")
	if err != nil {
		t.Fatalf("Error marshaling : %v", err)
	}
	t.Logf("%s", jsonPayload)

}
