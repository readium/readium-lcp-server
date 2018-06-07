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

import "regexp"

const (
	FIELD string = `<div class="mdl-textfield mdl-js-textfield">
						<label class="mdl-textfield__label">%s</label>
						<input name="%s" value="%s" %s class="mdl-textfield__input">
					</div>`
	FIELD_NO_LABEL string = `%s<input name="%s" value="%s" %s>`
	DATE_FIELD     string = `<div class="field">
 								<label>%s</label>
 								<input name="%s" id="%s" class="date_field" type="text" value="%s" data-date="%s" %s autocomplete="off">
 							</div>`
	TEXTAREA string = `<div class="field">
							<label>%s</label>
							<textarea name="%s" %s>%v</textarea>
						</div>`
	SELECT          string = `<select type="select" name="%s" id="%s">%s</select>`
	SELECT_NO_LABEL string = `%s<select type="select" name="%s" id="%s">
%s
</select>`
	SELECT_NO_ID string = `<div class="field">
<label>%s</label>
<select type="select" name="%s" %s>
%s
</select>
</div>`
	SELECT_NO_ID_NO_LABEL string = `%s<select type="select" name="%s" %s>
      %s
      </select>`
)

type (
	// Selectable provides an interface for options in a select
	Selectable interface {
		SelectName() string
		SelectValue() string
	}
	// SelectableOption provides a concrete implementation of Selectable - this should be called string option or similar
	SelectableOption struct {
		Name  string
		Value string
	}
	// Option type contains number and string
	Option struct {
		Id   int64
		Name string
	}
)

var (
	// Remove all other unrecognised characters apart from
	illegalName = regexp.MustCompile(`[^[:alnum:]-.]`)
	// We are very restrictive as this is intended for ascii url slugs
	illegalPath = regexp.MustCompile(`[^[:alnum:]\~\-\./]`)
	// Replace these separators with -
	baseNameSeparators = regexp.MustCompile(`[./]`)
	// A list of characters we consider separators in normal strings and replace with our canonical separator - rather than removing.

	separators = regexp.MustCompile(`[ &_=+:]`)

	dashes = regexp.MustCompile(`[\-]+`)
	// If the attribute contains data: or javascript: anywhere, ignore it
	// we don't allow this in attributes as it is so frequently used for xss
	// NB we allow spaces in the value, and lowercase.
	illegalAttr = regexp.MustCompile(`(d\s*a\s*t\s*a|j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*)\s*:`)

	// We are far more restrictive with href attributes.
	legalHrefAttr = regexp.MustCompile(`\A[/#][^/\\]?|mailto://|http://|https://`)

	ignoreTags = []string{"title", "script", "style", "iframe", "frame", "frameset", "noframes", "noembed", "embed", "applet", "object", "base"}

	defaultTags = []string{"h1", "h2", "h3", "h4", "h5", "h6", "div", "span", "hr", "p", "br", "b", "i", "strong", "em", "ol", "ul", "li", "a", "img", "pre", "code", "blockquote"}

	defaultAttributes = []string{"id", "class", "src", "href", "title", "alt", "name", "rel"}
	// A very limited list of transliterations to catch common european names translated to urls.
	// This set could be expanded with at least caps and many more characters.
	transliterations = map[rune]string{
		'À': "A",
		'Á': "A",
		'Â': "A",
		'Ã': "A",
		'Ä': "A",
		'Å': "AA",
		'Æ': "AE",
		'Ç': "C",
		'È': "E",
		'É': "E",
		'Ê': "E",
		'Ë': "E",
		'Ì': "I",
		'Í': "I",
		'Î': "I",
		'Ï': "I",
		'Ð': "D",
		'Ł': "L",
		'Ñ': "N",
		'Ò': "O",
		'Ó': "O",
		'Ô': "O",
		'Õ': "O",
		'Ö': "O",
		'Ø': "OE",
		'Ù': "U",
		'Ú': "U",
		'Ü': "U",
		'Û': "U",
		'Ý': "Y",
		'Þ': "Th",
		'ß': "ss",
		'à': "a",
		'á': "a",
		'â': "a",
		'ã': "a",
		'ä': "a",
		'å': "aa",
		'æ': "ae",
		'ç': "c",
		'è': "e",
		'é': "e",
		'ê': "e",
		'ë': "e",
		'ì': "i",
		'í': "i",
		'î': "i",
		'ï': "i",
		'ð': "d",
		'ł': "l",
		'ñ': "n",
		'ń': "n",
		'ò': "o",
		'ó': "o",
		'ô': "o",
		'õ': "o",
		'ō': "o",
		'ö': "o",
		'ø': "oe",
		'ś': "s",
		'ù': "u",
		'ú': "u",
		'û': "u",
		'ū': "u",
		'ü': "u",
		'ý': "y",
		'þ': "th",
		'ÿ': "y",
		'ż': "z",
		'Œ': "OE",
		'œ': "oe",
	}
)
