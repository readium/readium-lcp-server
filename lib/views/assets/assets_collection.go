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

// Package assets provides asset compilation, concatenation and fingerprinting.
package assets

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/logger"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Setup returns a new assets.Collection, also stores it in public var
func Setup(compiled bool, publicPath string, logger logger.StdLogger) AssetsCollection {
	Assets = AssetsCollection{
		serveCompiled: compiled,
		path:          publicPath + ASSETS,
		logger:        logger,
	}
	err := Assets.Load()
	if err != nil {
		logger.Errorf("Error loading assets %s\n", err)
	}
	return Assets
}

func RegisterAssetRoutes(muxer *mux.Router) {
	for _, g := range Assets.groups {
		for _, f := range g.files {
			route := ASSETS + strings.Replace(f.hash, "\\", "/", -1) + f.path
			//Assets.logger.Printf("Registering : %q", route)
			muxer.HandleFunc(route, func(w http.ResponseWriter, request *http.Request) {
				canonicalPath := path.Clean(request.URL.Path)
				// Try to find an asset in our list
				f := Assets.File(path.Base(canonicalPath))
				if f == nil {
					fmt.Printf("Asset not found %s base %s even if it was registered.\n", canonicalPath, path.Base(canonicalPath))
					return
				}
				if strings.Contains(f.name, ".js") {
					w.Header().Set("Content-Type", "application/javascript;charset=utf-8")
				} else if strings.Contains(f.name, ".css") {
					w.Header().Set("Content-Type", "text/css")
				}
				http.ServeContent(w, request, f.LocalPath(), time.Now(), bytes.NewReader(f.Bytes()))
			})
		}
	}
}

func PrintAssets() {
	for _, g := range Assets.groups {
		fmt.Printf("Name : %s\n Scripts : %s\nHash:%s\n", g.name, g.scripthash, g.stylehash)
		for _, f := range g.files {
			fmt.Printf("Path : %q Hash : %s\n", f.path, f.hash)
		}
	}
}

// File returns the first asset file matching name - this assumes files have unique names between groups
func (c *AssetsCollection) File(name string) *File {
	for _, g := range c.groups {
		for _, f := range g.files {
			//c.logger.Infof("Checking %s", f.name)
			if f.name == name {
				return f
			}
		}
	}
	return nil
}

// Group returns the named group if it exists or an empty group if not
func (c *AssetsCollection) Group(name string) *Group {
	for _, g := range c.groups {
		if g.name == name {
			return g
		}
	}
	return &Group{name: name} // Should this return nil instead?
}

// FetchOrCreateGroup returns the named group if it exists, or creates it if not
func (c *AssetsCollection) FetchOrCreateGroup(name string) *Group {
	for _, g := range c.groups {
		if g.name == name {
			return g
		}
	}
	g := &Group{name: name}
	c.groups = append(c.groups, g)
	return g
}

// MarshalJSON generates json for this collection, of the form {group:{file:hash}}
func (c *AssetsCollection) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("{")
	for i, g := range c.groups {
		gb, err := g.MarshalJSON()
		if err != nil {
			return nil, err
		}
		b.Write(gb)
		if i+1 < len(c.groups) {
			b.WriteString(",")
		}
	}
	b.WriteString("}")
	return b.Bytes(), nil
}

// Save the assets to a file after compilation
func (c *AssetsCollection) Save() error {
	// Get a representation of each file and group as json
	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return fmt.Errorf("Error marshalling assets file %s %v", c.path, err)
	}
	// Write our assets json file to the path
	err = ioutil.WriteFile(c.path, data, 0644)
	if err != nil {
		return fmt.Errorf("Error writing assets file %s %v", c.path, err)
	}
	return nil
}

// Load the asset groups from the assets json file
// Call this on startup from your app to read the asset details after assets are compiled
func (c *AssetsCollection) Load() error {
	// Make sure we reset groups, in case we compiled
	c.groups = make([]*Group, 0)
	// First scan the directory for files we're interested in
	files, err := c.collectAssets(filepath.Clean(c.path))
	if err != nil {
		return err
	}
	for _, f := range files {
		dir, file := filepath.Split(f)
		g := c.FetchOrCreateGroup(dir)
		g.AddAsset(file, dir)
		//c.logger.Infof("Group %s Asset %s \n", dir, file)
	}
	// For all our groups, sort files in name order
	for _, g := range c.groups {
		sort.Sort(g.files)
	}
	for _, g := range c.groups {
		for _, f := range g.files {
			//c.logger.Infof("Group %s Asset %s \n", g.name, f.path)
			f.bytes, err = ioutil.ReadFile(c.path + g.name + f.path)
			if err != nil {
				c.logger.Errorf("ERROR : %s\n", err)
				return err
			}
		}
	}

	return nil
}

// collectAssets collects the assets with this extension under src
func (c *AssetsCollection) collectAssets(src string) ([]string, error) {
	assetsExtensions := []string{".js", ".css", ".jpg", ".png", ".ico", ".map"}
	assets := []string{}
	//c.logger.Infof("Collecting assets from %s", src)

	err := filepath.Walk(src, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Deal with files, directories we return nil error to recurse on them
		if !info.IsDir() {
			for _, e := range assetsExtensions {
				if path.Ext(currentPath) == e {
					assets = append(assets, strings.Replace(currentPath, src, "", -1))
				}
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return assets, nil
	}
	return assets, nil
}

// bytesHash returns the sha hash of some bytes
func bytesHash(bytes []byte) string {
	sum := sha1.Sum(bytes)
	return hex.EncodeToString([]byte(sum[:]))
}
