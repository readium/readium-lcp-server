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

package http

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/readium/readium-lcp-server/lib/views/assets"
)

func (h *FileHandlerWrapper) ServeAsset(w http.ResponseWriter, request *http.Request) error {
	// Clean the path
	canonicalPath := path.Clean(request.URL.Path)
	// Try to find an asset in our list
	f := assets.Assets.File(path.Base(canonicalPath))
	if f == nil {
		if path.Base(canonicalPath) == "favicon.ico" {
			return nil
		}
		h.Log.Errorf("Asset not found %s base %s", canonicalPath, path.Base(canonicalPath))
		return fmt.Errorf("Asset not found %s in list", canonicalPath)
	}
	//context.Logf("Serving asset %s from RAM\n", f.LocalPath())
	http.ServeContent(w, request, f.LocalPath(), time.Now(), bytes.NewReader(f.Bytes()))
	return nil
}

// Default file handler, used in development - in production serve with nginx
func (h *FileHandlerWrapper) ServeFile(w http.ResponseWriter, request *http.Request) error {
	// Clean the path
	canonicalPath := path.Clean(request.URL.Path)
	// Assuming we're running from the root of the website
	localPath := h.Public + canonicalPath
	if _, err := os.Stat(localPath); err != nil {
		// If file not found return error
		if os.IsNotExist(err) {
			h.Log.Errorf("%s\n[serveFile] MISSING file %s\n", localPath, err.Error())
			return err
		}
		// For other errors return not authorised
		return fmt.Errorf("Not authorized access of %s", canonicalPath)
	}
	//fmt.Printf("Serving file %s\n", localPath)
	// If the file exists and we can access it, serve it
	http.ServeFile(w, request, localPath)
	return nil
}
