package storage

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSystemStorage(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "lcpserve_test_store", fmt.Sprintf("%d", rand.New(rand.NewSource(time.Now().UnixNano())).Int()))
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		t.Error("Could not create temp directory for test")
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(dir)

	store := NewFileSystem(dir, "http://localhost/assets")

	item, err := store.Add("test", bytes.NewReader([]byte("test1234")))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if item.Key() != "test" {
		t.Errorf("expected item key to be test, got %s", item.Key())
	}

	if item.PublicUrl() != "http://localhost/assets/test" {
		t.Errorf("expected item url to be http://localhost/assets/test, got %s", item.Key())
	}

	var buf [8]byte
	contents, err := item.Contents()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadFull(contents, buf[:]); err != nil {
		t.Fatal(err)
	} else {
		if string(buf[:]) != "test1234" {
			t.Error("expected buf to be test1234, got ", string(buf[:]))
		}
	}

	results, err := store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Error("Expected 1 element, got ", len(results))
	}

}

//func NewFileSystem(fs http.FileSystem, basePath string) storage.Store
