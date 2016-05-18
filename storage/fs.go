package storage

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type fsStorage struct {
	fspath string
	url    string
}

type fsItem struct {
	name       string
	storageDir string
	baseUrl    string
}

func (i fsItem) Key() string {
	return i.name
}

func (i fsItem) PublicUrl() string {
	return i.baseUrl + "/" + i.name
}

func (i fsItem) Contents() (io.ReadCloser, error) {
	return os.Open(filepath.Join(i.storageDir, i.name))
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

	return &fsItem{name: key, storageDir: s.fspath, baseUrl: s.url}, nil
}

func (s fsStorage) Get(key string) (Item, error) {
	_, err := os.Stat(filepath.Join(s.fspath, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NotFound
		} else {
			return nil, err
		}
	}

	return &fsItem{name: key, storageDir: s.fspath, baseUrl: s.url}, nil
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
		items = append(items, &fsItem{name: fi.Name(), storageDir: s.fspath, baseUrl: s.url})
	}

	return items, nil
}

func NewFileSystem(dir, basePath string) Store {
	return fsStorage{dir, basePath}
}
