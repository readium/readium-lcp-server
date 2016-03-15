package storage

import (
	"errors"
	"io"
)

var NotFound = errors.New("Item could not be found")

type Item interface {
	Key() string
	PublicUrl() string
	Contents() (io.ReadCloser, error)
}

type Store interface {
	Add(key string, r io.ReadSeeker) (Item, error)
	Get(key string) (Item, error)
	Remove(key string) error
	List() ([]Item, error)
}
