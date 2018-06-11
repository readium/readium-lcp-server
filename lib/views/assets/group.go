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

package assets

import (
	"bytes"
	"fmt"
	"github.com/readium/readium-lcp-server/lib/views/assets/minimizers/css"
	"github.com/readium/readium-lcp-server/lib/views/assets/minimizers/js"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func (a fileArray) Len() int {
	return len(a)
}

func (a fileArray) Less(b, c int) bool {
	return a[b].name < a[c].name
}

func (a fileArray) Swap(b, c int) {
	a[b], a[c] = a[c], a[b]
}

// Styles returns an array of file names for styles
func (g *Group) Styles() []*File {
	var styles []*File

	for _, f := range g.files {
		if f.Style() {
			styles = append(styles, f)
		}
	}

	return styles
}

// Scripts returns an array of file names for styles
func (g *Group) Scripts() []*File {
	var scripts []*File
	for _, f := range g.files {
		if f.Script() {
			scripts = append(scripts, f)
		}
	}
	return scripts
}

// RemoveFiles removes old compiled files for this group from dst
func (g *Group) RemoveFiles(dst string) error {
	if dst == "" {
		return fmt.Errorf("Empty destination string")
	}
	var assets []string
	pattern := path.Join(dst, ASSETS, SCRIPTS, g.name+"-*.min.js")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	assets = append(assets, files...)
	pattern = path.Join(dst, ASSETS, STYLES, g.name+"-*.min.css")
	files, err = filepath.Glob(pattern)
	if err != nil {
		return err
	}
	assets = append(assets, files...)
	for _, a := range assets {
		err = os.Remove(a)
		if err != nil {
			return err
		}
	}
	return nil
}

// Compile compiles all our files and calculates hashes from their contents
// The group hash is a hash of hashes
func (g *Group) Compile(dst string) error {
	scriptHashes, styleHashes := "", ""
	var scriptWriter, styleWriter bytes.Buffer
	for _, f := range g.files {
		if f.Script() {
			scriptHashes += f.hash
			scriptWriter.Write(f.bytes)
			scriptWriter.WriteString("\n\n")
		} else if f.Style() {
			styleHashes += f.hash
			styleWriter.Write(f.bytes)
			styleWriter.WriteString("\n\n")
		}
	}
	// Generate hashes for the files concatted using our existing file hashes as input
	// NB this is not the hash of the minified file
	g.scripthash = bytesHash([]byte(scriptHashes))
	g.stylehash = bytesHash([]byte(styleHashes))
	// Write out this group's minified concatted files
	err := g.writeFiles(dst, scriptWriter, styleWriter)
	// Reset the buffers on our files, which we no longer need
	for _, f := range g.files {
		f.bytes = nil
	}
	return err
}

// writeScript
func (g *Group) writeFiles(dst string, scriptWriter, styleWriter bytes.Buffer) error {
	var err error
	// Minify CSS
	miniCSS := css.Minify(styleWriter.Bytes())
	err = ioutil.WriteFile(g.StylePath(dst), miniCSS, PERMISSIONS)
	if err != nil {
		return err
	}
	// Minify JS
	minijs, err := js.Minify(scriptWriter.Bytes())
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(g.ScriptPath(dst), minijs, PERMISSIONS)
	if err != nil {
		return err
	}
	// Now reset our bytes buffers
	scriptWriter.Reset()
	styleWriter.Reset()
	return nil
}

// AddAsset adds this asset to the group
func (g *Group) AddAsset(p, h string) {
	file := &File{name: path.Base(p), path: p, hash: h}
	g.files = append(g.files, file)
}

// ParseFile adds this asset to our list of files, along with a fingerprint based on the content
func (g *Group) ParseFile(p string, dst string) error {
	// Create the file
	file, err := NewFile(p)
	if err != nil {
		return err
	}
	g.files = append(g.files, file)
	return nil
}

// String returns a string represention of group
func (g *Group) String() string {
	return fmt.Sprintf("%s:%d", g.name, len(g.files))
}

// StyleName returns a fingerprinted group name for styles
func (g *Group) StyleName() string {
	return fmt.Sprintf("%s-%s.min.css", g.name, g.stylehash)
}

// StylePath returns a fingerprinted group path for styles
func (g *Group) StylePath(dst string) string {
	return path.Join(dst, ASSETS, STYLES, g.StyleName())
}

// ScriptName returns a fingerprinted group name for scripts
func (g *Group) ScriptName() string {
	return fmt.Sprintf("%s-%s.min.js", g.name, g.scripthash)
}

// ScriptPath returns a fingerprinted group path for scripts
func (g *Group) ScriptPath(dst string) string {
	return path.Join(dst, ASSETS, SCRIPTS, g.ScriptName())
}

// MarshalJSON generates json for this collection, of the form {group:{file:hash}}
func (g *Group) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf(`"%s":{"scripts":"%s","styles":"%s","files":{`, g.name, g.scripthash, g.stylehash))
	for i, f := range g.files {
		fb, err := f.MarshalJSON()
		if err != nil {
			return nil, err
		}
		b.Write(fb)
		if i+1 < len(g.files) {
			b.WriteString(",")
		}
	}
	b.WriteString("}}")
	return b.Bytes(), nil
}
