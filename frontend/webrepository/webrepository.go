// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package webrepository

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	"path/filepath"

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

// Contains all repository definitions
type RepositoryManager struct {
	MasterRepositoryPath    string
	EncryptedRepositoryPath string
}

// Returns a specific repository file
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

// Returns all repository files
func (repManager RepositoryManager) GetMasterFiles() func() (RepositoryFile, error) {
	files, err := ioutil.ReadDir(repManager.MasterRepositoryPath)
	var fileIndex int

	if err != nil {
		return func() (RepositoryFile, error) { return RepositoryFile{}, err }
	}

	return func() (RepositoryFile, error) {
		var repFile RepositoryFile

		// Filter on epub
		for fileIndex < len(files) {
			file := files[fileIndex]
			fileExt := filepath.Ext(file.Name())
			fileIndex++

			if fileExt == ".epub" {
				repFile.Name = file.Name()
				return repFile, err
			}
		}

		return repFile, ErrNotFound
	}
}

// Open returns a WebPublication interface (db interaction)
func Init(config config.FrontendServerInfo) (i WebRepository, err error) {
	i = RepositoryManager{config.MasterRepository, config.EncryptedRepository}
	return
}
