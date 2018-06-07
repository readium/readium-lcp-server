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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Save the file represented by io.Reader to disk at path
func Save(r io.Reader, path string) error {
	// Write out to the desired file path
	w, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error - %s", err)
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	return err

}

// Given a file path, create all directories enclosing this file path (which may not yet exist)
func CreatePathTo(s string) error {
	if len(s) == 0 {
		return errors.New("Null path")
	}
	// Ignore the end of path, which is assumed to be a file
	s = filepath.Dir(s)
	s = filepath.Clean(s)
	fmt.Printf("Creating dirs to path %s\n", s)
	// Create all directories up to path
	return os.MkdirAll(s, PERMISSIONS)
}

// SanitizeName makes a string safe to use in a file name by first finding the path basename, then replacing non-ascii characters.
func SanitizeName(s string) string {
	// Start with lowercase string
	fileName := strings.ToLower(s)
	fileName = path.Clean(path.Base(fileName))
	// Remove illegal characters for names, replacing some common separators with -
	fileName = SanitizeString(fileName, illegalName)
	// NB this may be of length 0, caller must check
	return fileName
}

// SanitizeString replaces separators with - and removes characters listed in the regexp provided from string. Accents, spaces, and all characters not in A-Za-z0-9 are replaced.
func SanitizeString(s string, r *regexp.Regexp) string {
	// Remove any trailing space to avoid ending on -
	s = strings.Trim(s, " ")
	// Flatten accents first so that if we remove non-ascii we still get a legible name
	s = RemoveAccents(s)
	// Replace certain joining characters with a dash
	s = separators.ReplaceAllString(s, "-")
	// Remove all other unrecognised characters - NB we do allow any printable characters
	s = r.ReplaceAllString(s, "")
	// Remove any multiple dashes caused by replacements above
	s = dashes.ReplaceAllString(s, "-")
	return s
}

// RemoveAccents replaces a set of accented characters with ascii equivalents.
func RemoveAccents(s string) string {
	// Replace some common accent characters
	b := bytes.NewBufferString("")
	for _, c := range s {
		// Check transliterations first
		if val, ok := transliterations[c]; ok {
			b.WriteString(val)
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// NewFile returns a new file object
func NewFile(p string) (*File, error) {
	// Load file from path to get bytes
	bytes, err := ioutil.ReadFile(p)
	if err != nil {
		return &File{}, err
	}
	// Calculate hash and save it
	file := &File{
		path:  p,
		name:  path.Base(p),
		hash:  bytesHash(bytes),
		bytes: bytes,
	}
	return file, nil
}

// Style returns true if this file is a CSS file
func (f *File) Style() bool {
	return strings.HasSuffix(f.name, CSS_EXT)
}

// Script returns true if this file is a js file
func (f *File) Script() bool {
	return strings.HasSuffix(f.name, JS_EXT)
}

// MarshalJSON generates json for this file, of the form {group:{file:hash}}
func (f *File) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	s := fmt.Sprintf("\"%s\":\"%s\"", f.path, f.hash)
	b.WriteString(s)
	return b.Bytes(), nil
}

// Newer returns true if file exists at path
func (f *File) Newer(dst string) bool {
	// Check mtimes
	stat, err := os.Stat(f.path)
	if err != nil {
		return false
	}
	srcM := stat.ModTime()
	stat, err = os.Stat(dst)
	// If the file doesn't exist, return true
	if os.IsNotExist(err) {
		return true
	}
	// Else check for other errors
	if err != nil {
		return false
	}
	dstM := stat.ModTime()
	return srcM.After(dstM)
}

// Copy our bytes to dstpath
func (f *File) Copy(dst string) error {
	err := ioutil.WriteFile(dst, f.bytes, PERMISSIONS)
	if err != nil {
		return err
	}
	return nil
}

// LocalPath returns the relative path of this file
func (f *File) LocalPath() string {
	return f.path
}

// AssetPath returns the path of this file within the assets folder
func (f *File) AssetPath(dst string) string {
	folder := STYLES
	if f.Script() {
		folder = SCRIPTS
	}
	return path.Join(dst, ASSETS, folder, f.name)
}

// String returns a string representation of this object
func (f *File) String() string {
	return fmt.Sprintf("%s:%s", f.name, f.hash)
}

func (f *File) Bytes() []byte {
	return f.bytes
}
