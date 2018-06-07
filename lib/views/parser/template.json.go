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
	"fmt"
	got "html/template"
	"io"
)

// Setup performs one-time setup before parsing templates
func (t *JSONTemplate) Setup(helpers FuncMap) error {
	mu.Lock()
	defer mu.Unlock()
	jsonTemplateSet = got.New("").Funcs(got.FuncMap(helpers))
	return nil
}

// CanParseFile returns true if this template can parse this file
func (t *JSONTemplate) CanParseFile(path string) bool {
	allowed := []string{JSON_GOT}
	return suffixes(path, allowed)
}

// NewTemplate returns a new JSONTemplate
func (t *JSONTemplate) NewTemplate(fullpath, path string) (Template, error) {
	template := new(JSONTemplate)
	template.fullpath = fullpath
	template.path = path
	return template, nil
}

// Parse the template
func (t *JSONTemplate) Parse() error {
	mu.Lock()
	defer mu.Unlock()
	err := t.BaseTemplate.Parse()
	// Add to our template set
	if jsonTemplateSet.Lookup(t.Path()) == nil {
		_, err = jsonTemplateSet.New(t.path).Parse(t.Source())
	} else {
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// ParseString parses a string template
func (t *JSONTemplate) ParseString(s string) error {
	mu.Lock()
	defer mu.Unlock()
	err := t.BaseTemplate.ParseString(s)
	// Add to our template set
	if jsonTemplateSet.Lookup(t.Path()) == nil {
		_, err = jsonTemplateSet.New(t.path).Parse(t.Source())
	} else {
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// Finalize the template set, called after parsing is complete
func (t *JSONTemplate) Finalize(templates map[string]Template) error {
	// Go html/template records dependencies both ways (child <-> parent)
	// tmpl.Templates() includes tmpl and children and parents
	// we only want includes listed as dependencies
	// so just do a simple search of parsed source instead
	// Search source for {{\s template "|`xxx`|" x }} pattern
	paths := templateInclude.FindAllStringSubmatch(t.Source(), -1)
	// For all includes found, add the template to our dependency list
	for _, p := range paths {
		d := templates[p[1]]
		if d != nil {
			t.dependencies = append(t.dependencies, d)
		}
	}
	return nil
}

// Render the template
func (t *JSONTemplate) Render(writer io.Writer, context map[string]interface{}) error {
	jsonMu.RLock()
	defer jsonMu.RUnlock()
	tmpl := jsonTemplateSet.Lookup(t.Path())
	return tmpl.Execute(writer, context)
}
