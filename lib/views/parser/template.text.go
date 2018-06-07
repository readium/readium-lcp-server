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
	"io"
	got "text/template"
)

// Setup runs before parsing templates
func (t *TextTemplate) Setup(helpers FuncMap) error {
	textTemplateSet = got.New("").Funcs(got.FuncMap(helpers))
	return nil
}

// CanParseFile returns true if this parser handles this file path?
func (t *TextTemplate) CanParseFile(path string) bool {
	allowed := []string{TEXT_GOT, CSV_GOT}
	return suffixes(path, allowed)
}

// NewTemplate returns a new template of this type
func (t *TextTemplate) NewTemplate(fullpath, path string) (Template, error) {
	template := new(TextTemplate)
	template.fullpath = fullpath
	template.path = path
	return template, nil
}

// Parse the template
func (t *TextTemplate) Parse() error {
	err := t.BaseTemplate.Parse()
	template := textTemplateSet.Lookup(t.Path())
	// Add to our template set
	if template == nil {
		template = textTemplateSet.New(t.path)
		_, err = template.Parse(t.Source())
		if err != nil {
			template.Parse(fmt.Sprintf("PARSE ERROR %s\n", err))
		}
	} else {
		template.Parse(fmt.Sprintf("Duplicate template:%s %s", t.Path(), t.Source()))
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// ParseString a string template
func (t *TextTemplate) ParseString(s string) error {
	err := t.BaseTemplate.ParseString(s)
	// Add to our template set
	if textTemplateSet.Lookup(t.Path()) == nil {
		_, err = textTemplateSet.New(t.path).Parse(t.Source())
	} else {
		err = fmt.Errorf("Duplicate template:%s %s", t.Path(), t.Source())
	}
	return err
}

// Finalize the template set, called after parsing is complete
// Record a list of dependent templates (for breaking caches automatically)
func (t *TextTemplate) Finalize(templates map[string]Template) error {
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

// Render renders the template
func (t *TextTemplate) Render(writer io.Writer, context map[string]interface{}) error {
	tmpl := t.goTemplate()
	if tmpl == nil {
		return fmt.Errorf("Error rendering template:%s %s", t.Path(), t.Source())
	}
	return tmpl.Execute(writer, context)
}

// goTemplate returns teh underlying go template
func (t *TextTemplate) goTemplate() *got.Template {
	return textTemplateSet.Lookup(t.Path())
}
