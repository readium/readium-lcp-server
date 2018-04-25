package store_test

import (
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/store"
	"testing"
	"time"
)

func TestTransactionEventStore_Add(t *testing.T) {
	licenses, err := stor.LicenseStatus().List(1, 1, 0)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading license statuses : %v", err)
	}
	if len(licenses) == 0 {
		t.Skip("Forgot to create license statuses?")
	}
	err = stor.Transaction().Add(
		&store.TransactionEvent{
			DeviceName:      faker.Name().Name(),
			Timestamp:       time.Now().UTC().Truncate(time.Second),
			Type:            store.StatusActive,
			LicenseStatusFk: licenses[0].Id,
		})
	if err != nil {
		t.Fatalf("Error creating transaction")
	}
}

func TestTransactionEventStore_GetByLicenseStatusId(t *testing.T) {
	licenses, err := stor.LicenseStatus().List(1, 1, 0)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading license statuses : %v", err)
	}
	if len(licenses) == 0 {
		t.Skip("Forgot to create license statuses?")
	}

	transactions, err := stor.Transaction().GetByLicenseStatusId(licenses[0].Id)
	if err != nil {
		t.Fatalf("Error reading transaction : %v", err)
	}
	for _, trans := range transactions {

		t.Logf("%#v", trans)
	}
}

func TestTransactionEventStore_Get(t *testing.T) {
	transaction, err := stor.Transaction().Get(1)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading transaction : %v", err)
	}
	t.Logf("%#v", transaction)
}

func TestTransactionEventStore_CheckDeviceStatus(t *testing.T) {
	licenses, err := stor.LicenseStatus().List(1, 1, 0)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading license statuses : %v", err)
	}

	licenseId := licenses[0].Id
	transColl, err := stor.Transaction().GetByLicenseStatusId(licenseId)
	if err != nil {
		t.Fatalf("Error reading transaction : %v", err)
	}
	if len(transColl) == 0 {
		t.Skip("Empty collection.")
	}
	deviceId := transColl[0].DeviceId

	stat, err := stor.Transaction().CheckDeviceStatus(licenseId, deviceId)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading transaction : %v", err)
	}
	t.Logf("Status : %#v", stat)
}
