package store_test

import (
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/store"
	"testing"
)

func TestPurchaseStore_Add(t *testing.T) {
	users, err := stor.User().List(30, 0)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading users : %v", err)
	}
	for _, user := range users {
		entity := &store.Purchase{
			User: user,
			Publication: &store.Publication{
				Title: faker.Name().Name(),
			},
		}
		v1 := faker.RandomInt64(0, 2)

		if v1 == 1 {
			entity.StartDate = store.TruncatedNow()
		}

		if v1 == 0 {
			entity.EndDate = store.TruncatedNow()
		}

		err = stor.Purchase().Add(entity)
		if err != nil {
			t.Fatalf("Error saving : %v", err)
		}

		t.Logf("Purchase created : %#v", entity)
	}
}
