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

// Package parser defines an interface for parsers (creating templates) and templates (rendering content), and defines a base template type which conforms to both interfaces and can be included in any templates
package parser

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// NewScanner creates a new template scanner
func NewScanner(paths []string, helpers FuncMap, serverLogger Logger) (*Scanner, error) {
	s := &Scanner{
		Helpers:   helpers,
		Paths:     paths,
		Templates: make(map[string]Template),
		//Parsers:      []Parser{new(JSONTemplate), new(HTMLTemplate), new(TextTemplate)},
		//TODO : enable if needed
		/** Disabled unused template types **/
		Parsers:      []Parser{new(HTMLTemplate)},
		serverLogger: serverLogger,
	}
	return s, nil
}

// ScanPath scans a path for template files, including sub-paths
func (s *Scanner) ScanPath(root string) error {
	//s.serverLogger.Printf("Scanning %s\n", root)
	root = path.Clean(root)
	// Store current path, and change to root path
	// so that template includes use relative paths from root
	// this may not be necc. any more, test removing it
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Chdir(root)
	if err != nil {
		return err
	}
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Deal with files, directories we return nil error to recurse on them
		if !info.IsDir() {
			// Ask parsers in turn to handle the file - first one to claim it wins
			for _, p := range s.Parsers {
				if p.CanParseFile(path) {
					fullpath := filepath.Join(root, path)
					path := strings.Replace(path, string(os.PathSeparator), "/", -1)
					t, err := p.NewTemplate(fullpath, path)
					if err != nil {
						return err
					}
					//s.serverLogger.Printf("Adding fullpath template %s to templates as '%s'", fullpath, path)
					s.Templates[path] = t
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Change back to original path
	err = os.Chdir(pwd)
	if err != nil {
		return err
	}
	return nil
}

// ScanPaths resets template list and rescans all template paths
func (s *Scanner) ScanPaths() error {
	// Make sure templates is empty
	s.Templates = make(map[string]Template)
	// Set up the parsers
	for _, p := range s.Parsers {
		err := p.Setup(s.Helpers)
		if err != nil {
			return err
		}
	}
	// Scan paths again
	for _, p := range s.Paths {
		err := s.ScanPath(p)
		if err != nil {
			return err
		}
	}
	// Now parse and finalize templates
	for _, t := range s.Templates {
		err := t.Parse()
		if err != nil {
			s.serverLogger.Printf("Error : %s", err)
		}
	}
	// Now finalize templates
	for _, t := range s.Templates {
		err := t.Finalize(s.Templates)
		if err != nil {
			return err
		}
	}
	return nil
}

// PATH UTILITIES

// dotFile returns true if the file path supplied a dot file?
func dotFile(p string) bool {
	return strings.HasPrefix(path.Base(p), ".")
}

// suffix returns true if the path have this suffix (ignoring dotfiles)?
func suffix(p string, suffix string) bool {
	if dotFile(p) {
		return false
	}
	return strings.HasSuffix(p, suffix)
}

// suffixes returns true if the path has these suffixes (ignoring dotfiles)?
func suffixes(p string, suffixes []string) bool {
	if dotFile(p) {
		return false
	}
	for _, s := range suffixes {
		if strings.HasSuffix(p, s) {
			return true
		}
	}
	return false
}
