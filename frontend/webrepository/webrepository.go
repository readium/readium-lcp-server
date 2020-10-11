// Copyright (c) 2020 Readium Foundation
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package webrepository

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	"github.com/readium/readium-lcp-server/config"
)

// ErrNotFound error trown when repository is not found
var ErrNotFound = errors.New("Repository not found")

// WebRepository interface for repository db interaction
type WebRepository interface {
	GetMasterFile(name string) (RepositoryFile, error)
	GetMasterFiles() func() (RepositoryFile, error)
}

// RepositoryFile struct defines a file stored in a repository
type RepositoryFile struct {
	Name string `json:"name"`
	Path string
}

// RepositoryManager contains all repository definitions
type RepositoryManager struct {
	MasterRepositoryPath    string
	EncryptedRepositoryPath string
}

// GetMasterFile returns a specific repository file
func (repManager RepositoryManager) GetMasterFile(name string) (RepositoryFile, error) {
	var filePath = path.Join(repManager.MasterRepositoryPath, name)

	if _, err := os.Stat(filePath); err == nil {
		// File exists
		var repFile RepositoryFile
		repFile.Name = name
		repFile.Path = filePath
		return repFile, err
	}

	return RepositoryFile{}, ErrNotFound
}

// GetMasterFiles returns all filenames from the master repository
func (repManager RepositoryManager) GetMasterFiles() func() (RepositoryFile, error) {
	files, err := ioutil.ReadDir(repManager.MasterRepositoryPath)
	var fileIndex int

	if err != nil {
		return func() (RepositoryFile, error) { return RepositoryFile{}, err }
	}

	return func() (RepositoryFile, error) {
		var repFile RepositoryFile

		for fileIndex < len(files) {
			file := files[fileIndex]
			repFile.Name = file.Name()
			fileIndex++
			return repFile, err
		}

		return repFile, ErrNotFound
	}
}

// Init returns a WebPublication interface (db interaction)
func Init(config config.FrontendServerInfo) (i WebRepository, err error) {
	i = RepositoryManager{config.MasterRepository, config.EncryptedRepository}
	return
}
