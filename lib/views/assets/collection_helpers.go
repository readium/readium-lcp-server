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
	"fmt"
	"html/template"
	"strings"
)

// StyleLink converts a set of group names to one style link tag (production) or to a list of style link tags (development)
func (c *AssetsCollection) StyleLink(names ...string) template.HTML {
	var html template.HTML
	// Iterate through names, setting up links for each
	// we link to groups if we have them, else we fall back to normal links
	for _, name := range names {
		g := c.Group(name)
		if g.stylehash != "" {
			if c.serveCompiled {
				html = html + StyleLink(g.StyleName())
			} else {
				for _, f := range g.Styles() {
					html = html + StyleLink(f.name) + template.HTML("\n")
				}
			}
		} else {
			html = html + StyleLink(name)
		}
	}
	return html
}

// ScriptLink converts a set of group names to one script tag (production) or to a list of script tags (development)
func (c *AssetsCollection) ScriptLink(names ...string) template.HTML {
	var html template.HTML
	// Iterate through names, setting up links for each
	// we link to groups if we have them, else we fall back to normal links
	for _, name := range names {
		g := c.Group(name)
		if g.stylehash != "" {
			if c.serveCompiled {
				html = html + ScriptLink(g.ScriptName())
			} else {
				for _, f := range g.Scripts() {
					html = html + ScriptLink(f.name) + template.HTML("\n")
				}
			}
		} else {
			html = html + ScriptLink(name)
		}
	}
	return html
}

// StyleLink returns an html tag for a given file path
func StyleLink(name string) template.HTML {
	if !strings.HasSuffix(name, CSS_EXT) {
		name = name + CSS_EXT
	}
	return template.HTML(fmt.Sprintf(styleTemplate, template.URLQueryEscaper(name)))
}

// ScriptLink returns an html tag for a given file path
func ScriptLink(name string) template.HTML {
	if !strings.HasSuffix(name, JS_EXT) {
		name = name + JS_EXT
	}
	return template.HTML(fmt.Sprintf(scriptTemplate, template.URLQueryEscaper(name)))
}
