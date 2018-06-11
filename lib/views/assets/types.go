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
	log "github.com/readium/readium-lcp-server/lib/logger"
	"regexp"
)

const (
	SCRIPTS string = "scripts"
	STYLES  string = "styles"
	FILES   string = "files"
	ASSETS  string = "/assets"

	CSS_EXT        string = ".css"
	JS_EXT         string = ".js"
	styleTemplate         = `<link href="/assets/styles/%s" media="all" rel="stylesheet" type="text/css" />`
	scriptTemplate        = `<script src="/assets/js/%s" type="text/javascript" ></script>`
	PERMISSIONS           = 0744
)

type (
	// Collection holds the complete list of groups
	AssetsCollection struct {
		serveCompiled bool
		path          string
		groups        []*Group
		logger        log.StdLogger
	}
	// A sortable file array
	fileArray []*File
	// File stores a filename and hash fingerprint for the asset file
	File struct {
		name  string
		hash  string
		path  string
		bytes []byte
	}
	// Group holds a name and a list of files (images, scripts, styles)
	Group struct {
		name       string
		files      fileArray
		stylehash  string // the hash of the compiled group css file (if any)
		scripthash string // the hash of the compiled group js file (if any)
	}
	// Options represents image options
	Options struct {
		Path      string
		MaxHeight int64
		MaxWidth  int64
		Quality   int
		// Square bool
	}
)

var (
	Assets AssetsCollection
	// Remove all other unrecognised characters apart from
	illegalName = regexp.MustCompile(`[^[:alnum:]-.]`)
	// A list of characters we consider separators in normal strings and replace with our canonical separator - rather than removing.
	separators = regexp.MustCompile(`[ &_=+:]`)

	dashes = regexp.MustCompile(`[\-]+`)
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
