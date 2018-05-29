/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package filestor

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type (
	fsStorage struct {
		fspath string
		url    string
	}

	fsItem struct {
		name       string
		storageDir string
		baseURL    string
	}
)

func (i fsItem) Key() string {
	return i.name
}

func (i fsItem) PublicURL() string {
	return i.baseURL + "/" + i.name
}

func (i fsItem) Contents() (io.ReadCloser, error) {
	return os.Open(filepath.Join(i.storageDir, i.name))
	// FIXME: process errors
}

func (s fsStorage) Add(key string, r io.ReadSeeker) (Item, error) {
	file, err := os.Create(filepath.Join(s.fspath, key))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	io.Copy(file, r)

	if err != nil {
		return nil, err
	}
	return &fsItem{name: key, storageDir: s.fspath, baseURL: s.url}, nil
}

// Get returns an Item in the storage, by its key
// the key is the file name
//
func (s fsStorage) Get(key string) (Item, error) {
	_, err := os.Stat(filepath.Join(s.fspath, key))
	if err != nil {
		if os.IsNotExist(err) {
			//println(s.fspath + " " + key + " does not exist.")
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &fsItem{name: key, storageDir: s.fspath, baseURL: s.url}, nil
}

func (s fsStorage) Remove(key string) error {
	return os.Remove(filepath.Join(s.fspath, key))
}

func (s fsStorage) List() ([]Item, error) {
	var items []Item

	files, err := ioutil.ReadDir(s.fspath)
	if err != nil {
		return nil, err
	}

	for _, fi := range files {
		items = append(items, &fsItem{name: fi.Name(), storageDir: s.fspath, baseURL: s.url})
	}

	return items, nil
}

// NewFileSystem creates a new storage
//
func NewFileSystem(dir, basePath string) Store {
	return fsStorage{dir, basePath}
}
