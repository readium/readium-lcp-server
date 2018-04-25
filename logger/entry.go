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
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

func NewEntry(logger *Logger) *Entry {
	return &Entry{
		Logger: logger,
		// Default is three fields, give a little extra room
		Data: make(Fields, 5),
	}
}

// Returns a reader for the entry, which is a proxy to the formatter.
func (entry *Entry) Reader() (*bytes.Buffer, error) {
	serialized, err := entry.Logger.formatter.Format(entry)
	return bytes.NewBuffer(serialized), err
}

// SetFormatter
func (entry *Entry) SetFormatter(formatter Formatter) {
	entry.Logger.Lock()
	defer entry.Logger.Unlock()
	entry.Logger.formatter = formatter
}

// SetOutput
func (entry *Entry) SetOutput(output io.Writer) {
	entry.Logger.Lock()
	defer entry.Logger.Unlock()
	entry.Logger.output = output
}

// SetLevel
func (entry *Entry) SetLevel(level Level) {
	entry.Logger.Lock()
	defer entry.Logger.Unlock()
	entry.Logger.level = level
}

// Debug formated
func (entry *Entry) Debugf(format string, args ...interface{}) {
	if entry.Logger.level >= DebugLevel {
		entry.log(DebugLevel, fmt.Sprintf(format, args...))
	}
}

// Info formated
func (entry *Entry) Infof(format string, args ...interface{}) {
	if entry.Logger.level >= InfoLevel {
		entry.log(InfoLevel, fmt.Sprintf(format, args...))
	}
}

// Warn Formated
func (entry *Entry) Warnf(format string, args ...interface{}) {
	if entry.Logger.level >= WarnLevel {
		entry.log(WarnLevel, fmt.Sprintf(format, args...))
	}
}

// Error Formated
func (entry *Entry) Errorf(format string, args ...interface{}) {
	if entry.Logger.level >= ErrorLevel {
		entry.log(ErrorLevel, fmt.Sprintf(format, args...))
	}
}

// Fatal Formated
func (entry *Entry) Fatalf(format string, args ...interface{}) {
	if entry.Logger.level >= FatalLevel {
		entry.log(FatalLevel, fmt.Sprintf(format, args...))
	}
	os.Exit(1)
}

// Panic Formated
func (entry *Entry) Panicf(format string, args ...interface{}) {
	if entry.Logger.level >= PanicLevel {
		entry.log(PanicLevel, fmt.Sprintf(format, args...))
	}
}

// Panic Formated
func (entry *Entry) Printf(format string, args ...interface{}) {
	entry.log(entry.Logger.level, fmt.Sprintf(format, args...))
}

// Add a map of fields to the Entry.
func (entry *Entry) WithFields(fields Fields) *Entry {
	data := make(Fields, len(entry.Data)+len(fields))
	for k, v := range entry.Data {
		data[k] = v
	}
	for k, v := range fields {
		data[k] = v
	}
	return &Entry{Logger: entry.Logger, Data: data}
}

// Actual loggin subroutine
func (entry Entry) log(level Level, msg string) {
	entry.Time = time.Now()
	entry.Level = level
	entry.Message = msg

	reader, err := entry.Reader()
	if err != nil {
		entry.Logger.Lock()
		fmt.Fprintf(os.Stderr, "Failed to obtain reader, %v\n", err)
		entry.Logger.Unlock()
	}

	entry.Logger.Lock()
	defer entry.Logger.Unlock()

	_, err = io.Copy(entry.Logger.output, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
	}

	// To avoid Entry#log() returning a value that only would make sense for
	// panic() to use in Entry#Panic(), we avoid the allocation by checking
	// directly here.
	if level <= PanicLevel {
		panic(&entry)
	}
}
