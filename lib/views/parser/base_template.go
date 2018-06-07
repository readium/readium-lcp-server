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

package parser

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
)

// TODO - base template is a mixin, get rid of all methods which are going to be overridden like StartParse
// PARSER

// Setup sets up the template for parsing
func (t *BaseTemplate) Setup(viewsPath string, helpers FuncMap) error {
	return nil
}

// CanParseFile returns true if we can parse this file
func (t *BaseTemplate) CanParseFile(path string) bool {
	if dotFile(path) {
		return false
	}
	return true
}

// NewTemplate returns a newly created template for this path
func (t *BaseTemplate) NewTemplate(fullpath, path string) (Template, error) {
	template := new(BaseTemplate)
	template.fullpath = fullpath
	template.path = path
	return template, nil
}

// TEMPLATE PARSING

// Parse the template (BaseTemplate simply stores it)
func (t *BaseTemplate) Parse() error {
	// Read the file
	s, err := t.readFile(t.fullpath)
	if err == nil {
		t.source = s
	}
	return err
}

// ParseString a string template
func (t *BaseTemplate) ParseString(s string) error {
	t.path = t.generateHash(s)
	t.source = s
	return nil
}

// Render the template ignoring context
func (t *BaseTemplate) Render(writer io.Writer, context map[string]interface{}) error {
	writer.Write([]byte(t.Source()))
	return nil
}

// Finalize is called on each template after parsing is finished, supplying complete template set.
func (t *BaseTemplate) Finalize(templates map[string]Template) error {
	t.dependencies = []Template{}
	return nil
}

// Source the parsed version of this template
func (t *BaseTemplate) Source() string {
	return t.source
}

// Path returns the path of this template
func (t *BaseTemplate) Path() string {
	return t.path
}

// CacheKey returns the cache key of this template -
// (this is generated from path + hash of contents + dependency hash keys).
// So it automatically changes when templates are changed
func (t *BaseTemplate) CacheKey() string {
	// If we have a key, return it
	// NB this relies on templates being reloaded on reload of app in production...
	if t.key != "" {
		return t.key
	}
	//println("Making key for",t.Path())
	// Otherwise generate the key
	t.key = t.path + "/" + t.generateHash(t.Source())
	for _, d := range t.dependencies {
		t.key = t.key + "-" + d.CacheKey()
	}
	// Finally, if the key is too long, set it to a hash of the key instead
	// (Memcache for example has limits on key length)
	// possibly we should deal with this at a higher level
	// I'd suggest always md5 keys with /views/ prefix...
	// put this into cache itself though...
	if len(t.key) > MaxCacheKeyLength {
		t.key = t.generateHash(t.key)
	}
	return t.key
}

// Dependencies returns which other templates this one depends on (for generating nested cache keys)
func (t *BaseTemplate) Dependencies() []Template {
	return t.dependencies
}

// Utility method to read files into a string
func (t *BaseTemplate) readFile(path string) (string, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		println("Error reading template file at path ", path)
		return "", err
	}
	return string(fileBytes), err
}

// Utility method to generate a hash from string
func (t *BaseTemplate) generateHash(input string) string {
	// FIXME: use sha256, not md5
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}
