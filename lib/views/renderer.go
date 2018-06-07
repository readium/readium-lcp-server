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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/readium/readium-lcp-server/lib/views/parser"
	"html/template"
	"io"
	"net/http"
	"path"
)

func Mock(viewPath string, rendererContext map[string]interface{}) *Renderer {
	return &Renderer{
		status:        http.StatusOK,
		renderContext: rendererContext,
		view:          viewPath,
	}
}
func MockWithDefaultLayout(viewPath string, defaultLayoutPath string, rendererContext map[string]interface{}) *Renderer {
	return &Renderer{
		status:        http.StatusOK,
		renderContext: rendererContext,
		view:          viewPath,
		layout:        defaultLayoutPath,
	}
}

type RendererConfig struct {
	ViewPath          string
	Path              string
	DefaultLayoutPath string
	ContentType       string
	RenderContext     map[string]interface{}
	Writer            http.ResponseWriter
}

func (r *Renderer) SetWriter(w http.ResponseWriter) {
	r.writer = w
}

// Layout sets the layout used
func (r *Renderer) Layout(layout string) *Renderer {
	r.layout = layout
	return r
}

// Template sets the template used
func (r *Renderer) Template(template string) *Renderer {
	r.view = template
	return r
}

func (r *Renderer) GetTemplate() string {
	return r.view
}

// Path sets the request path on the renderer (used for choosing a default template)
func (r *Renderer) Path(p string) *Renderer {
	r.path = path.Clean(p)
	return r
}

// Status returns the Renderer status
func (r *Renderer) Status(status int) *Renderer {
	r.status = status
	return r
}

// Text sets the views content as text
func (r *Renderer) Text(content string) *Renderer {
	if r.renderContext == nil {
		r.renderContext = make(map[string]interface{})
	}
	r.renderContext["content"] = content
	return r
}

// HTML sets the views content as html (use with caution)
func (r *Renderer) HTML(content string) *Renderer {
	if r.renderContext == nil {
		r.renderContext = make(map[string]interface{})
	}
	r.renderContext["content"] = template.HTML(content)
	return r
}

// AddKey adds a key/value pair to context
func (r *Renderer) AddKey(key string, value interface{}) *Renderer {
	if r.renderContext == nil {
		r.renderContext = make(map[string]interface{})
	}
	r.renderContext[key] = value
	return r
}

// Context sets the entire context for rendering
func (r *Renderer) Context(c map[string]interface{}) *Renderer {

	if r.renderContext == nil {
		r.renderContext = make(map[string]interface{})
	}
	r.renderContext = c
	return r
}

func (r *Renderer) RetrieveTemplate() parser.Template {
	mu.RLock()
	t := scanner.Templates[r.view]
	mu.RUnlock()
	return t
}

func (r *Renderer) RetrieveLayout() parser.Template {
	mu.RLock()
	t := scanner.Templates[r.layout]
	mu.RUnlock()
	return t
}

// RenderToString renders our template into layout using our context and return a string
func (r *Renderer) RenderToString() (string, error) {
	content := ""
	if len(r.view) > 0 {
		t := r.RetrieveTemplate()
		if t == nil {
			return content, fmt.Errorf("No such template found %s", r.view)
		}
		var rendered bytes.Buffer
		err := t.Render(&rendered, r.renderContext)
		if err != nil {
			return content, err
		}
		content = rendered.String()
	}
	return content, nil
}

// FIXME - test for side-effects then replace RenderToString with the layout version as a bug fix

// RenderToStringWithLayout renders our template into layout using our context and return a string
func (r *Renderer) RenderToStringWithLayout() (string, error) {
	var rendered bytes.Buffer
	// We require a template
	if len(r.view) > 0 {
		t := r.RetrieveTemplate()
		if t == nil {
			return "", fmt.Errorf("No such template found %s", r.view)
		}
		// Render the template to a buffer
		err := t.Render(&rendered, r.renderContext)
		if err != nil {
			return "", err
		}
		// Render that buffer into the layout if we have one
		if len(r.layout) > 0 {
			r.renderContext["content"] = template.HTML(rendered.String())
			l := r.RetrieveLayout()
			if l == nil {
				return "", fmt.Errorf("No such layout found %s", r.layout)
			}
			// Render the layout to the buffer
			rendered.Reset()
			err := l.Render(&rendered, r.renderContext)
			if err != nil {
				return "", err
			}
		}
	}
	return rendered.String(), nil
}

// Render our template into layout using our context and write out to writer
func (r *Renderer) Render(values ...interface{}) error {
	if r.layout == "" {
		r.layout = DefaultLayoutPath
	}
	if r.contentType == "application/json" {
		if len(values) == 1 {
			bytes, jsonErr := json.Marshal(values[0])
			if jsonErr != nil {
				return jsonErr
			}
			//directly to writer
			_, err := io.WriteString(r.writer, string(bytes))
			return err
		} else {
			_, err := io.WriteString(r.writer, "ok")
			return err
		}
	}
	// Reload if not in production
	if !Production {
		//r.debug("Reloading templates in development mode")
		err := ReloadTemplates()
		if err != nil {
			return err
		}
	}
	// If we have a template, render it
	// using r.Context unless overridden by content being set with .Text("My string")
	if len(r.view) > 0 && r.renderContext["content"] == nil {
		t := r.RetrieveTemplate()
		if t == nil {
			return fmt.Errorf("#error No such template found %s", r.view)
		}
		var rendered bytes.Buffer
		err := t.Render(&rendered, r.renderContext)
		if err != nil {
			return fmt.Errorf("#error Could not render template %s - %s", r.view, err)
		}
		if r.layout != "" {
			r.renderContext["content"] = template.HTML(rendered.String())
		} else {
			r.renderContext["content"] = rendered.String()
		}
	}
	// Now render the content into the layout template
	if r.layout != "" {
		layout := r.RetrieveLayout()
		if layout == nil {
			return fmt.Errorf("#error Could not find layout %s", r.layout)
		}
		err := layout.Render(r.writer, r.renderContext)
		if err != nil {
			return fmt.Errorf("#error Could not render layout %s %s", r.layout, err)
		}
	} else if r.renderContext["content"] != nil {
		// Deal with no layout by rendering content directly to writer
		_, err := io.WriteString(r.writer, r.renderContext["content"].(string))
		return err
	}
	return nil
}

//DISABLED
/**
func (r *Renderer) getTemplate(view string) parser.Template {
	mu.RLock()
	defer mu.RUnlock()

	if r.view != "" {
		// Try template view
		tmfile := filepath.Join(r.view, view)
		tm, found := scanner.Templates[tmfile]
		if found {
			return tm
		}
	}

	// Try default template view
	dflfile := filepath.Join(scanner.Default, view)
	td, found := scanner.Templates[dflfile]
	if found {
		return td
	}

	return nil
}
**/
