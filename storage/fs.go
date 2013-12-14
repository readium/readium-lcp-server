package storage

import (
  "os"
  "io"
  "path/filepath"
)

type fsStorage struct {
  fspath string
  url string
}

type fsItem struct {
  f *os.File
  name string
  base string
}

func (i fsItem) Key() string {
  return i.name
}

func (i fsItem) PublicUrl() string {
  return i.base + "/" + i.name
}

func (i fsItem) Contents() io.Reader {
  return i.f
}


func (s fsStorage) Add(key string, r io.Reader) (Item, error) {
  file, err := os.Create(filepath.Join(s.fspath, key))
  defer file.Seek(0, 0)
  io.Copy(file, r)

  if err != nil {
    return nil, err
  }

  return &fsItem{file, key, s.url}, nil
}

func (s fsStorage) Get(key string) (Item, error) {
  file, err := os.Open(filepath.Join(s.fspath, key))
  if err != nil {
    return nil, NotFound
  }
  return &fsItem{file, key, s.url}, nil
}

func (s fsStorage) Remove(key string) error {
  return os.Remove(filepath.Join(s.fspath, key))
}

func (s fsStorage) List() Iterator {
  i := 0
  size := 0
  var items []os.FileInfo
  d, err := os.Open(s.fspath)
  if err == nil {
    items, err = d.Readdir(0)
    if err == nil {
      size = len(items)
    }
  }
  return func() (Item, error) {
    if i < size {
      i++
      return s.Get(items[i - 1].Name())
    } else {
      return nil, io.EOF
    }
  }
}

func NewFileSystem(dir, basePath string) Store {
 return fsStorage{dir, basePath}
}
