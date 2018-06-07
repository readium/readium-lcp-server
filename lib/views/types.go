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

package views

import (
	"github.com/readium/readium-lcp-server/lib/views/parser"
	"net/http"
	"sync"
)

type (
	Debugf func(format string, args ...interface{})
	// Renderer is a views which is set up on each request and renders the response to its writer
	Renderer struct {
		// The views rendering context
		renderContext map[string]interface{}
		// The writer to write the context to
		writer http.ResponseWriter
		// The layout template to render in
		layout string
		// The template to render
		view string
		// The status to render with
		status int
		// The request path
		path string
		// content type
		contentType string
	}
	// RenderContext is the type passed in to New, which helps construct the rendering views
	// Alternatively, you can use NewWithPath, which doesn't require a RenderContext
	RenderContext interface {
		Path() string
		RenderContext() map[string]interface{}
		Writer() http.ResponseWriter
		DefaultLayoutPath() string
		ViewPath() string
		ContentType() string
	}
)

var (
	// Production is true if this server is running in production mode
	Production bool

	DefaultLayoutPath string
	// The scanner is a private type used for scanning templates
	scanner *parser.Scanner
	// This mutex guards the pkg scanner variable during reload and access
	// it is only neccessary because of hot reload during development
	mu sync.RWMutex
	// Helper functions available in templates
	Helpers parser.FuncMap
)
