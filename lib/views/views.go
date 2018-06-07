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

// Package view provides methods for rendering templates, and helper functions for golang views
package views

import (
	"fmt"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/lib/views/assets"
	"github.com/readium/readium-lcp-server/lib/views/helpers"
	"github.com/readium/readium-lcp-server/lib/views/parser"
	"time"
)

// TODO remove public pkg vars completely, except perhaps Production

func init() {
	Helpers = DefaultHelpers()
}

// DefaultHelpers returns a default set of helpers for the app, which can then be extended/replaced
// NB if you change helper functions the templates must be reloaded at least once afterwards
func DefaultHelpers() parser.FuncMap {

	/**
		builtins = FuncMap{
	    		"and":      and,
	    		"call":     call,
	    		"html":     HTMLEscaper,
	    		"index":    index,
	    		"js":       JSEscaper,
	    		"len":      length,
	    		"not":      not,
	    		"or":       or,
	    		"print":    fmt.Sprint,
	    		"printf":   fmt.Sprintf,
	    		"println":  fmt.Sprintln,
	    		"urlquery": URLQueryEscaper,

	    		// Comparisons
	    		"eq": eq, // ==
	    		"ge": ge, // >=
	    		"gt": gt, // >
	    		"le": le, // <=
	    		"lt": lt, // <
	    		"ne": ne, // !=
	    	}
	*/
	return parser.FuncMap{
		// HEAD helpers
		"style":          helpers.Style,
		"script":         helpers.Script,
		"externalscript": helpers.ExternalScript,
		"dev": func() bool {
			return !Production
		},
		// HTML helpers
		"html":     helpers.HTML,
		"htmlattr": helpers.HTMLAttribute,
		"url":      helpers.URL,
		"sanitize": helpers.Sanitize,
		"strip":    helpers.Strip,
		"truncate": helpers.Truncate,
		// XML helpers
		"xmlpreamble": helpers.XMLPreamble,
		// Form helpers
		"int64":            helpers.Int64FromString,
		"field":            helpers.Field,
		"datefield":        helpers.DateField,
		"textarea":         helpers.TextArea,
		"select":           helpers.Select,
		"disabledSelect":   helpers.DisabledSelect,
		"selectarray":      helpers.SelectArray,
		"genericselect":    helpers.GenericSelect,
		"optionsforselect": helpers.OptionsForSelect,
		"utcdate":          helpers.UTCDate,
		"utctime":          helpers.UTCTime,
		"utcnow":           helpers.UTCNow,
		"date":             helpers.Date,
		"time":             helpers.Time,
		"numberoptions":    helpers.NumberOptions,
		// CSV helper
		"csv": helpers.CSV,
		// String helpers
		"blank":  helpers.Blank,
		"exists": helpers.Exists,
		// Math helpers
		"mod":      helpers.Mod,
		"odd":      helpers.Odd,
		"add":      helpers.Add,
		"subtract": helpers.Subtract,
		// Array functions
		"array":    helpers.Array,
		"append":   helpers.Append,
		"contains": helpers.Contains,
		// Map functions
		"map":      helpers.Map,
		"set":      helpers.Set,
		"setif":    helpers.SetIf,
		"empty":    helpers.Empty,
		"last":     helpers.Last,
		"isEmpty":  helpers.IsEmpty,
		"notEmpty": helpers.NotEmpty,
		"defined":  helpers.Defined,
		// Numeric helpers - clean up and accept currency and other options in centstoprice
		"centstobase":         helpers.CentsToBase,
		"centstoprice":        helpers.CentsToPrice,
		"centstopriceshort":   helpers.CentsToPriceShort,
		"pricetocents":        helpers.PriceToCents,
		"dynamicValue":        helpers.DynamicScopeValue,
		"dynamicBooleanValue": helpers.DynamicScopeBooleanValue,
		"dynamicStringValue":  helpers.DynamicStringValue,

		"displayFloat": helpers.FormatFloatNoDecimals,
	}
}

// LoadTemplatesAtPaths loads our templates given the paths provided
func LoadTemplatesAtPaths(paths []string, helpers parser.FuncMap, logger logger.StdLogger) error {
	mu.Lock()
	defer mu.Unlock()
	// Scan all templates within the given paths, using the helpers provided
	var err error
	scanner, err = parser.NewScanner(paths, helpers, logger)
	if err != nil {
		return err
	}
	err = scanner.ScanPaths()
	if err != nil {
		return err
	}
	return nil
}

// ReloadTemplates reloads the templates for our scanner
func ReloadTemplates() error {
	//TODO : panic recoverer, since if it's crashing server stops
	mu.Lock()
	defer mu.Unlock()
	return scanner.ScanPaths()
}

// PrintTemplates prints out our list of templates for debug
func PrintTemplates() {
	mu.RLock()
	defer mu.RUnlock()
	for k := range scanner.Templates {
		fmt.Printf("Template %s\n", k)
	}
}
func HasTemplate(staticPath string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for k := range scanner.Templates {
		if k == staticPath {
			return true
		}
	}
	return false
}

func SetupView(logger logger.StdLogger, isProduction, areAssetsCompiled bool, publicPath string) {
	// Compilation of assets is done on deploy
	// We just load them here
	appAssets := assets.Setup(areAssetsCompiled, publicPath, logger)
	// Set up helpers which are aware of fingerprinted assets
	// These behave differently depending on the compile flag above
	// when compile is set to no, they use precompiled assets
	// otherwise they serve all files in a group separately
	Helpers["style"] = appAssets.StyleLink
	Helpers["script"] = appAssets.ScriptLink
	logger.Infof("Finished loading assets in %s\n", time.Now())
	Production = isProduction
	//s.ErrorLogger.Debugf("SERVER URL %s", ServerURL)
	err := LoadTemplatesAtPaths([]string{publicPath}, Helpers, logger)
	//display all registred templates
	//views.PrintTemplates()
	if err != nil {
		logger.Errorf("Error reading templates %s\n", err)
	}
	logger.Infof("Finished loading templates in %s\n", time.Now())
}
