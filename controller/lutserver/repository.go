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

package lutserver

import (
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/views"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type (
	// RepositoryFile struct defines a file stored in a repository
	RepositoryFile struct {
		Name string `json:"name"`
		Path string
	}
	// Contains all repository definitions
	RepositoryManager struct {
		MasterRepositoryPath    string
		EncryptedRepositoryPath string
	}
)

var (
	repoManager RepositoryManager
)

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
		var result RepositoryFile

		// Filter on epub
		for fileIndex < len(files) {
			file := files[fileIndex]
			fileExt := filepath.Ext(file.Name())
			fileIndex++

			if fileExt == ".epub" {
				result.Name = file.Name()
				return result, err
			}
		}

		return result, ErrNotFound
	}
}

// GetRepositoryMasterFiles returns a list of repository masterfiles
func GetRepositoryMasterFiles(server http.IServer) (*views.Renderer, error) {
	files := make([]RepositoryFile, 0)
	fn := repoManager.GetMasterFiles()

	for it, err := fn(); err == nil; it, err = fn() {
		files = append(files, it)
	}
	view := &views.Renderer{}
	view.AddKey("files", files)
	view.AddKey("masterRepo", server.Config().LutServer.MasterRepository)
	view.AddKey("encryptRepo", server.Config().LutServer.EncryptedRepository)
	view.AddKey("pageTitle", "Repositories")
	view.Template("repository/index.html.got")

	return view, nil
}
