package storage

import (
  "testing"
  "os"
  "bytes"
  "io"
)

func TestFileSystemStorage(t * testing.T) {
  store := NewFileSystem(os.TempDir(), "http://localhost/assets")

  item, err := store.Add("test", bytes.NewBufferString("test1234"));
  if  err != nil {
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
  if _, err = io.ReadFull(item.Contents(), buf[:]); err != nil {
    t.Error(err)
    t.FailNow()
  } else {
    if string(buf[:]) != "test1234" {
      t.Error("expected buf to be test1234, got ", string(buf[:]))
    }
  }
  
}


//func NewFileSystem(fs http.FileSystem, basePath string) storage.Store
