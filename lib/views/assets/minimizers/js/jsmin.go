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

package js

// ... or should I say, a direct braindead port of the ugly Crockford's code...

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const eof = -1

type minifier struct {
	buf          *bytes.Buffer
	r            *bufio.Reader
	w            *bufio.Writer
	theA         int
	theB         int
	theLookahead int
	theX         int
	theY         int
	err          error
}

func (m *minifier) init(r *bufio.Reader, w *bufio.Writer) {
	m.r = r
	m.w = w
	m.theLookahead = eof
	m.theX = eof
	m.theY = eof
}

func (m *minifier) error(s string) error {
	m.err = fmt.Errorf("JSMIN Error: %s", s)
	return m.err
}

/* isAlphanum -- return true if the character is a letter, digit, underscore,
   dollar sign, or non-ASCII character.
*/

func isAlphanum(c int) bool {
	return ((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') || c == '_' || c == '$' || c == '\\' ||
		c > 126)
}

/* get -- return the next character from stdin. Watch out for lookahead. If
   the character is a control character, translate it to a space or
   linefeed.
*/

func (m *minifier) get() int {
	c := m.theLookahead
	m.theLookahead = eof
	if c == eof {
		b, err := m.r.ReadByte()
		if err != nil {
			if err == io.EOF {
				c = eof
			} else {
				m.error(err.Error())
				return eof
			}
		} else {
			c = int(b)
		}
	}
	if c >= ' ' || c == '\n' || c == eof {
		return c
	}
	if c == '\r' {
		return '\n'
	}
	return ' '
}

/* peek -- get the next character without getting it.
 */

func (m *minifier) peek() int {
	m.theLookahead = m.get()
	return m.theLookahead
}

/* next -- get the next character, excluding comments. peek() is used to see
   if a '/' is followed by a '/' or '*'.
*/

func (m *minifier) next() int {
	c := m.get()
	if c == '/' {
		switch m.peek() {
		case '/':
			for {
				c = m.get()
				if c <= '\n' {
					break
				}
			}
		case '*':
			m.get()
			// Preserve license comments (/*!)
			if m.peek() == '!' {
				m.get()
				m.putc('/')
				m.putc('*')
				m.putc('!')
				for c != 0 {
					c = m.get()
					switch c {
					case '*':
						if m.peek() == '/' {
							m.get()
							c = 0
						}
						break
					case eof:
						m.error("Unterminated comment.")
						return eof
					default:
						m.putc(c)
					}
				}
				m.putc('*')
				m.putc('/')
			}
			// --
			for c != ' ' {
				switch m.get() {
				case '*':
					if m.peek() == '/' {
						m.get()
						c = ' '
					}
					break
				case eof:
					m.error("Unterminated comment.")
					return eof
				}
			}
		}
	}
	m.theY = m.theX
	m.theX = c
	return c
}

/* action -- do something! What you do is determined by the argument:
        1   Output A. Copy B to A. Get the next B.
        2   Copy B to A. Get the next B. (Delete A).
        3   Get the next B. (Delete B).
   action treats a string as a single character. Wow!
   action recognizes a regular expression if it is preceded by ( or , or =.
*/
func (m *minifier) putc(c int) {
	m.w.WriteByte(byte(c))
}

func (m *minifier) action(d int) {
	switch d {
	case 1:
		m.putc(m.theA)
		if (m.theY == '\n' || m.theY == ' ') &&
			(m.theA == '+' || m.theA == '-' || m.theA == '*' || m.theA == '/') &&
			(m.theB == '+' || m.theB == '-' || m.theB == '*' || m.theB == '/') {
			m.putc(m.theY)
		}
		fallthrough
	case 2:
		m.theA = m.theB
		if m.theA == '\'' || m.theA == '"' || m.theA == '`' {
			for {
				m.putc(m.theA)
				m.theA = m.get()
				if m.theA == m.theB {
					break
				}
				if m.theA == '\\' {
					m.putc(m.theA)
					m.theA = m.get()
				}
				if m.theA == eof {
					m.error("Unterminated string literal.")
					return
				}
			}
		}
		fallthrough
	case 3:
		m.theB = m.next()
		if m.theB == '/' && (m.theA == '(' || m.theA == ',' || m.theA == '=' || m.theA == ':' ||
			m.theA == '[' || m.theA == '!' || m.theA == '&' || m.theA == '|' ||
			m.theA == '?' || m.theA == '+' || m.theA == '-' || m.theA == '~' ||
			m.theA == '*' || m.theA == '/' || m.theA == '{' || m.theA == '\n') {
			m.putc(m.theA)
			if m.theA == '/' || m.theA == '*' {
				m.putc(' ')
			}
			m.putc(m.theB)
			for {
				m.theA = m.get()
				if m.theA == '[' {
					for {
						m.putc(m.theA)
						m.theA = m.get()
						if m.theA == ']' {
							break
						}
						if m.theA == '\\' {
							m.putc(m.theA)
							m.theA = m.get()
						}
						if m.theA == eof {
							m.error("Unterminated set in Regular Expression literal.")
							return
						}
					}
				} else if m.theA == '/' {
					switch m.peek() {
					case '/', '*':
						m.error("Unterminated set in Regular Expression literal.")
						return
					}
					break
				} else if m.theA == '\\' {
					m.putc(m.theA)
					m.theA = m.get()
				}
				if m.theA == eof {
					m.error("Unterminated Regular Expression literal.")
					return
				}
				m.putc(m.theA)
			}
			m.theB = m.next()
		}
	}
}

/* jsmin -- Copy the input to the output, deleting the characters which are
   insignificant to JavaScript. Comments will be removed. Tabs will be
   replaced with spaces. Carriage returns will be replaced with linefeeds.
   Most spaces and linefeeds will be removed.
*/

func (m *minifier) run() {
	if m.peek() == 0xEF {
		m.get()
		m.get()
		m.get()
	}
	m.theA = '\n'
	m.action(3)
	for m.theA != eof {
		switch m.theA {
		case ' ':
			if isAlphanum(m.theB) {
				m.action(1)
			} else {
				m.action(2)
			}
		case '\n':
			switch m.theB {
			case '{', '[', '(', '+', '-', '!', '~':
				m.action(1)
			case ' ':
				m.action(3)
			default:
				if isAlphanum(m.theB) {
					m.action(1)
				} else {
					m.action(2)
				}
			}
		default:
			switch m.theB {
			case ' ':
				if isAlphanum(m.theA) {
					m.action(1)
				} else {
					m.action(3)
				}
			case '\n':
				switch m.theA {
				case '}', ']', ')', '+', '-', '"', '\'', '`':
					m.action(1)
				default:
					if isAlphanum(m.theA) {
						m.action(1)
					} else {
						m.action(3)
					}
				}
			default:
				m.action(1)
			}
		}
	}
}

// Minify returns a minified script or an error.
func Minify(script []byte) (minified []byte, err error) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	r := bufio.NewReader(bytes.NewReader(script))

	m := new(minifier)
	m.init(r, w)
	m.run()
	if m.err != nil {
		return nil, err
	}
	w.Flush()

	minified = buf.Bytes()
	if len(minified) > 0 && minified[0] == '\n' {
		minified = minified[1:]
	}
	return minified, nil
}
