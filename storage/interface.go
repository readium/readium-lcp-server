package storage

import (
  "io"
  "errors"
)

var NotFound = errors.New("Item could not be found")

type Item interface {
  Key() string
  PublicUrl() string
  Contents() io.Reader
}

type Iterator func() (Item, error)

type Store interface {
  Add(key string, r io.Reader) (Item, error)
  Get(key string) (Item, error)
  Remove(key string) error
  List() Iterator
}
