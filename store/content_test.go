package store_test

import (
	"testing"

	"bytes"
	"crypto/sha256"
	"github.com/dmgk/faker"
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/store"
	"github.com/satori/go.uuid"
)

func TestContentStore_Add(t *testing.T) {
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	hasher := sha256.New()
	for i := 0; i < 30; i++ {
		// generate a key
		key, err := encrypter.GenerateKey()
		if err != nil {
			t.Fatalf("Error generating encryption key : %v", err)
		}
		// content
		content := &store.Content{
			EncryptionKey: key,
			Location:      faker.App().Name(),
			Length:        faker.RandomInt64(0, 65535),
			Sha256:        string(hasher.Sum(nil)),
		}
		// insert in the database
		err = stor.Content().Add(content)
		if err != nil {
			t.Fatalf("Error creating content : %v", err)
		}
	}
}

func TestContentStore_Get(t *testing.T) {
	// reading all
	contents, err := stor.Content().List()
	if err != nil {
		t.Fatalf("Error retrieving contents : %v", err)
	}
	if len(contents) == 0 {
		t.Skipf("You forgot to create contents?")
	}
	// choosing first content UUID
	content, err := stor.Content().Get(contents[0].Id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving content by uuid : %v", err)
		}
	}
	t.Logf("Found content by uuiid : %#v", content)
}

func TestContentStore_List(t *testing.T) {
	contents, err := stor.Content().List()
	if err != nil {
		t.Fatalf("Error retrieving contents : %v", err)
	}
	for idx, content := range contents {
		t.Logf("%d. %#v", idx, content)
	}
}

func TestContentStore_Update(t *testing.T) {
	// reading all
	contents, err := stor.Content().List()
	if err != nil {
		t.Fatalf("Error retrieving contents : %v", err)
	}
	if len(contents) == 0 {
		t.Skipf("You forgot to create contents?")
	}
	// choosing first content UUID
	content, err := stor.Content().Get(contents[0].Id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving content by uuid : %v", err)
		}
	}
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	// generate a new key
	key, err := encrypter.GenerateKey()
	if err != nil {
		t.Fatalf("Error generating encryption key : %v", err)
	}
	// generate a new uuid
	newUUID, errU := uuid.NewV4()
	if errU != nil {
		t.Fatalf("Error generating new uuid : %v", errU)
	}

	// set the new key and save
	content.EncryptionKey = key
	content.Id = newUUID.String()
	err = stor.Content().Update(content)
	if err != nil {
		t.Fatalf("Error updating content : %v", err)
	}
	// read it again (by updated uuid) to compare encryption key
	updatedContent, err := stor.Content().Get(newUUID.String())
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("Error retrieving content by uuid : %v", err)
		}
	}
	// verify information was saved
	if !bytes.Equal(updatedContent.EncryptionKey, key) {
		t.Fatalf("Error : encryption key was not updated.")
	}
	t.Logf("Updated content's encryption key")
}
