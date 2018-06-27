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

package epub

import (
	"archive/zip"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func (epub *Epub) addCleartextResources(names []string) {
	if epub.cleartextResources == nil {
		epub.cleartextResources = []string{}
	}

	for _, name := range names {
		epub.cleartextResources = append(epub.cleartextResources, name)
	}
}

func (epub *Epub) addCleartextResource(name string) {
	if epub.cleartextResources == nil {
		epub.cleartextResources = []string{}
	}

	epub.cleartextResources = append(epub.cleartextResources, name)
}

// io.Writer implementation
func (epub Epub) Write(dst io.Writer) error {
	writer := NewWriter(dst)

	err := writer.WriteHeader()
	if err != nil {
		return err
	}

	for _, res := range epub.Resource {
		if res.Path != "mimetype" {
			fw, err := writer.AddResource(res.Path, res.StorageMethod)
			if err != nil {
				return err
			}
			_, err = io.Copy(fw, res.Contents)
			if err != nil {
				return err
			}
		}
	}

	if epub.Encryption != nil {
		writeEncryption(epub, writer)
	}

	return writer.Close()
}

func (epub Epub) Cover() (bool, *Resource) {

	for _, p := range epub.Package {

		var coverImageID string
		coverImageID = "cover-image"
		for _, meta := range p.Metadata.Metas {
			if meta.Name == "cover" {
				coverImageID = meta.Content
			}
		}

		for _, it := range p.Manifest.Items {

			if strings.Contains(it.Properties, "cover-image") ||
				it.Id == coverImageID {

				path := filepath.Join(p.BasePath, it.Href)
				for _, r := range epub.Resource {
					if r.Path == path {
						return true, r
					}
				}
			}
		}
	}

	return false, nil
}

func (epub *Epub) Add(name string, body io.Reader, size uint64) error {
	epub.Resource = append(epub.Resource, &Resource{Contents: body, StorageMethod: zip.Deflate, Path: name, OriginalSize: size})

	return nil
}

func (epub Epub) CanEncrypt(file string) bool {
	i := sort.SearchStrings(epub.cleartextResources, file)
	return i >= len(epub.cleartextResources) || epub.cleartextResources[i] != file
}
