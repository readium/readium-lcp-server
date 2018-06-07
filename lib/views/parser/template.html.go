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

// Setup performs setup before parsing templates
func (t *HTMLTemplate) Setup(helpers FuncMap) error {
	mu.Lock()
	defer mu.Unlock()
	htmlTemplateSet = got.New("").Funcs(got.FuncMap(helpers))
	return nil
}

// CanParseFile returns true if this parser handles this path
func (t *HTMLTemplate) CanParseFile(path string) bool {
	allowed := []string{HTML_GOT, XML_GOT}
	return suffixes(path, allowed)
}

// NewTemplate returns a new template for this type
func (t *HTMLTemplate) NewTemplate(fullpath, path string) (Template, error) {
	template := new(HTMLTemplate)
	template.fullpath = fullpath
	template.path = path
	return template, nil
}

// Parse the template at path
func (t *HTMLTemplate) Parse() error {
	mu.Lock()
	defer mu.Unlock()
	err := t.BaseTemplate.Parse()
	if err != nil {
		return err
	}
	template := htmlTemplateSet.Lookup(t.Path())
	// Add to our template set - NB duplicates not allowed by golang templates
	if template == nil {
		template = htmlTemplateSet.New(t.path)
		_, err = template.Parse(t.Source())
		if err != nil {
			template.Parse(fmt.Sprintf("PARSE ERROR: %s \n", err))
		}
	} else {
		template.Parse(fmt.Sprintf("Duplicate template:%s %s", t.Path(), t.Source()))
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// ParseString parses a string template
func (t *HTMLTemplate) ParseString(s string) error {
	mu.Lock()
	defer mu.Unlock()
	err := t.BaseTemplate.ParseString(s)
	// Add to our template set
	if htmlTemplateSet.Lookup(t.Path()) == nil {
		_, err = htmlTemplateSet.New(t.path).Parse(t.Source())
	} else {
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// Finalize the template set, called after parsing is complete
func (t *HTMLTemplate) Finalize(templates map[string]Template) error {
	// Go html/template records dependencies both ways (child <-> parent)
	// tmpl.Templates() includes tmpl and children and parents
	// we only want includes listed as dependencies
	// so just do a simple search of unparsed source instead
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

// Render the template to the given writer, returning an error
func (t *HTMLTemplate) Render(writer io.Writer, context map[string]interface{}) error {
	mu.RLock()
	defer mu.RUnlock()
	tmpl := htmlTemplateSet.Lookup(t.Path())
	if tmpl == nil {
		return fmt.Errorf("#error loading template for %s", t.Path())
	}
	return tmpl.Execute(writer, context)
}
