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
	html "html/template"
	"io"
	"regexp"
	"sync"
	text "text/template"
)

const (
	JSON_GOT string = ".json.got"
	HTML_GOT string = ".html.got"
	XML_GOT  string = ".xml.got"
	TEXT_GOT string = ".text.got"
	CSV_GOT  string = ".csv.got"
)

type (
	Logger interface {
		Printf(format string, args ...interface{})
	}
	// FuncMap is a map of functions
	FuncMap map[string]interface{}
	// Parser loads template files, and returns a template suitable for rendering content
	Parser interface {
		// Setup is called once on setup of a parser
		Setup(helpers FuncMap) error
		// Can this parser handle this file?
		CanParseFile(path string) bool
		// Parse the file given and return a compiled template
		NewTemplate(fullpath, path string) (Template, error)
	}
	// Scanner scans paths for templates and creates a representation of each using parsers
	Scanner struct {
		// A map of all templates keyed by path name
		Templates map[string]Template
		// A set of parsers (in order) with which to parse templates
		Parsers []Parser
		// A set of paths (in order) from which to load templates
		Paths []string
		// Helpers is a list of helper functions
		Helpers FuncMap
		// Default template
		Default      string
		serverLogger Logger
	}
	// Template renders its content given a ViewContext
	Template interface {
		// Parse a template file
		Parse() error
		// Called after parsing is finished
		Finalize(templates map[string]Template) error
		// Render to this writer
		Render(writer io.Writer, context map[string]interface{}) error
		// Return the original template content
		Source() string
		// Return the template path
		Path() string
		// Return the cache key
		CacheKey() string
		// Return dependencies of this template (used for creating cache keys)
		Dependencies() []Template
	}
	// BaseTemplate is a base template which conforms to Template and Parser interfaces.
	// This is an abstract base type, we use html or text templates
	BaseTemplate struct {
		fullpath     string     // the full true path from project root
		path         string     // the relative template path from src - used for unique identifier
		source       string     // at present we store in memory
		key          string     // set at parse time
		dependencies []Template // set at parse time
	}
	// HTMLTemplate represents an HTML template using go HTML/template
	HTMLTemplate struct {
		BaseTemplate
	}
	// JSONTemplate represents a template using go HTML/template
	JSONTemplate struct {
		BaseTemplate
	}
	// TextTemplate using go text/template
	TextTemplate struct {
		BaseTemplate
	}
)

var (
	templateInclude = regexp.MustCompile(`{{\s*template\s*["]([\S]*)["].*}}`)

	// MaxCacheKeyLength determines the max key length for cache keys
	MaxCacheKeyLength = 250
	mu                sync.RWMutex // Shared mutex to go with shared template set, because of dev reloads
	jsonMu            sync.RWMutex // Shared mutex to go with shared template set, because of dev reloads

	htmlTemplateSet *html.Template // This is a shared template set for HTML templates
	jsonTemplateSet *html.Template // This is a shared template set for json templates
	textTemplateSet *text.Template
)
