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

package helpers

import (
	"fmt"
	"strings"

	got "html/template"
)

// Style inserts a css tag
func Style(name string) got.HTML {
	return got.HTML(fmt.Sprintf("<link href=\"/assets/styles/%s.css\" media=\"all\" rel=\"stylesheet\" type=\"text/css\" />", EscapeURL(name)))
}

// Script inserts a script tag
func Script(name string) got.HTML {
	return got.HTML(fmt.Sprintf("<script src=\"/assets/js/%s.js\" type=\"text/javascript\"></script>", EscapeURL(name)))
}

func ExternalScript(url string) got.HTML {
	return got.HTML(fmt.Sprintf("<script src=\"%s.js\" type=\"text/javascript\"></script>", url))
}

// Escape escapes HTML using HTMLEscapeString
func Escape(s string) string {
	return got.HTMLEscapeString(s)
}

// EscapeURL escapes URLs using HTMLEscapeString
func EscapeURL(s string) string {
	return got.URLQueryEscaper(s)
}

// Link returns got.HTML with an anchor link given text and URL required
// Attributes (if supplied) should not contain user input
func Link(t string, u string, a ...string) got.HTML {
	attributes := ""
	if len(a) > 0 {
		attributes = strings.Join(a, " ")
	}
	return got.HTML(fmt.Sprintf("<a href=\"%s\" %s>%s</a>", Escape(u), Escape(attributes), Escape(t)))
}

// HTML returns a string (which must not contain user input) as go template HTML
func HTML(s string) got.HTML {
	return got.HTML(s)
}

// HTMLAttribute returns a string (which must not contain user input) as go template HTMLAttr
func HTMLAttribute(s string) got.HTMLAttr {
	return got.HTMLAttr(s)
}

// URL returns returns a string (which must not contain user input) as go template URL
func URL(s string) got.URL {
	return got.URL(s)
}

// Strip all html tags and returns as go template HTML
func Strip(s string) got.HTML {
	return got.HTML(sanitizeHTML(s))
}

// Sanitize the html, leaving only tags we consider safe (see the sanitize package for details and tests)
func Sanitize(s string) got.HTML {
	s, err := sanitizeHTMLAllowing(s)
	if err != nil {
		fmt.Printf("#error sanitizing html:%s", err)
		return got.HTML("")
	}
	return got.HTML(s)
}

// XMLPreamble returns an XML preamble as got.HTML,
// primarily to work around a bug in html/template which escapes <?
// see https://github.com/golang/go/issues/12496
func XMLPreamble() got.HTML {
	return `<?xml version="1.0" encoding="UTF-8"?>`
}
