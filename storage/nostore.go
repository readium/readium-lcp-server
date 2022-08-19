// Copyright (c) 2022 Readium Foundation
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package storage

import (
	"io"
)

// void storage, created to avoid breaking interfaces in case the storage is handled by the encryption tool.

type noStorage struct {
}

type noItem struct {
	name string
}

// noItem functions

func (i noItem) Key() string {
	return i.name
}

func (i noItem) PublicURL() string {
	return ""
}

func (i noItem) Contents() (io.ReadCloser, error) {
	return nil, ErrNotFound
}

// noStorage functions

func (s noStorage) Add(key string, r io.ReadSeeker) (Item, error) {
	return &noItem{name: key}, nil
}

func (s noStorage) Get(key string) (Item, error) {
	return nil, ErrNotFound
}

func (s noStorage) Remove(key string) error {
	return ErrNotFound
}

func (s noStorage) List() ([]Item, error) {
	return nil, ErrNotFound
}

// NoStorage creates a new void storage
//
func NoStorage() Store {
	return noStorage{}
}
