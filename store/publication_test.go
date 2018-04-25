package store_test

import (
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/store"
	"testing"
)

func TestPublicationStore_Add(t *testing.T) {
	for i := 0; i < 100; i++ {
		pub := &store.Publication{
			Title: faker.Name().Name(),
		}
		err := stor.Publication().Add(pub)
		if err != nil {
			t.Fatalf("Error creating publication : %v", err)
		}
	}
}

func TestPublicationStore_Get(t *testing.T) {
	publication, err := stor.Publication().Get(1)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving publication by id : %v", err)
		} else {
			t.Skipf("You forgot to create publication with id = 1 ?")
		}
	}
	t.Logf("Found publication with id 1 : %#v", publication)
}

func TestPublicationStore_GetByUUID(t *testing.T) {
	publication, err := stor.Publication().GetByUUID("")
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving publication by UUID : %v", err)
		} else {
			t.Skipf("You forgot to create publication with UUID `` ?")
		}
	}
	t.Logf("Found publication with UUID `` : %#v", publication)
}

func TestPublicationStore_CheckByTitle(t *testing.T) {
	checkedTitle := "Jillian Kovaceku"
	counter, err := stor.Publication().CheckByTitle(checkedTitle)
	if err != nil {
		t.Fatalf("Error checking publications : %v", err)
	}
	t.Logf("%d counter with title %q", counter, checkedTitle)
}

func TestPublicationStore_List(t *testing.T) {
	publications, err := stor.Publication().List(30, 0)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	for idx, pub := range publications {
		t.Logf("%d. %#v", idx, pub)
	}

	publications, err = stor.Publication().List(50, 1)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	for idx, pub := range publications {
		t.Logf("%d. %#v", idx, pub)
	}
}

func TestPublicationStore_Update(t *testing.T) {

}

func TestPublicationStore_Delete(t *testing.T) {
	pubs, err := stor.Publication().List(1, 0)
	if err != nil {
		t.Fatalf("Error retrieving publications : %v", err)
	}
	if len(pubs) != 1 {
		t.Fatalf("No users found. Create some publications.")
	}
	// deleting the first publication
	err = stor.Publication().Delete(pubs[0].ID)
	if err != nil {
		t.Fatalf("Error deleting Publication : %v", err)
	}
	t.Log("Publication deleted : %#v", pubs[0])
}
