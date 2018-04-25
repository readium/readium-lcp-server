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

package logger

import (
	"io"
	"os"
)

var (
	_ StdLogger = &Logger{}
	_ StdLogger = &Entry{}
)

//
func New() StdLogger {
	return &Logger{
		output:    os.Stderr,
		level:     InfoLevel,
		formatter: &TextFormatter{},
	}
}

// SetFormatter sets the standard logger formatter.
func (logger *Logger) SetFormatter(formatter Formatter) {
	NewEntry(logger).SetFormatter(formatter)
}

// SetLevel sets the standard logger level.
func (logger *Logger) SetLevel(level Level) {
	NewEntry(logger).SetLevel(level)
}

// SetOutput set the output interface
func (logger *Logger) SetOutput(output io.Writer) {
	NewEntry(logger).SetOutput(output)
}

// Debug Formated Message
func (logger *Logger) Debugf(format string, args ...interface{}) {
	NewEntry(logger).Debugf(format, args...)
}

// Info Formated Message
func (logger *Logger) Infof(format string, args ...interface{}) {
	NewEntry(logger).Infof(format, args...)
}

// Warning formated Message
func (logger *Logger) Warnf(format string, args ...interface{}) {
	NewEntry(logger).Warnf(format, args...)
}

// Error Formated Message
func (logger *Logger) Errorf(format string, args ...interface{}) {
	NewEntry(logger).Errorf(format, args...)
}

// Fatal Formated Message
func (logger *Logger) Fatalf(format string, args ...interface{}) {
	NewEntry(logger).Fatalf(format, args...)
}

// Panic Formated Message
func (logger *Logger) Panicf(format string, args ...interface{}) {
	NewEntry(logger).Panicf(format, args...)
}

// General log
func (logger *Logger) Printf(format string, args ...interface{}) {
	NewEntry(logger).Printf(format, args...)
}

// Log with Fields
func (logger *Logger) WithFields(fields Fields) *Entry {
	return NewEntry(logger).WithFields(fields)
}

// Open log file
func Open(logfile string, mode uint32) (*os.File, error) {

	lf, err := os.OpenFile(
		logfile,
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		os.FileMode(mode),
	)

	if err != nil {
		return nil, err
	}

	return lf, nil
}
