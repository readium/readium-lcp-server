package model_test

import (
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/model"
	"testing"
)

func TestPurchaseStore_Add(t *testing.T) {
	users, err := stor.User().List(30, 0)
	if err != nil && err != gorm.ErrRecordNotFound {
		t.Fatalf("Error reading users : %v", err)
	}
	for _, user := range users {
		entity := &model.Purchase{
			User: *user,
			Publication: model.Publication{
				Title: faker.Name().Name(),
			},
		}
		v1 := faker.RandomInt64(0, 2)

		if v1 == 1 {
			entity.StartDate = model.TruncatedNow()
		}

		if v1 == 0 {
			entity.EndDate = model.TruncatedNow()
		}

		err = stor.Purchase().Add(entity)
		if err != nil {
			t.Fatalf("Error saving : %v", err)
		}

		t.Logf("Purchase created : %#v", entity)
	}
}
