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
	"sync"
	"time"
)

type (
	Level uint8

	// used for Log.WithFieds
	Fields map[string]interface{}

	StdLogger interface {
		Debugf(format string, args ...interface{})
		Infof(format string, args ...interface{})
		Warnf(format string, args ...interface{})
		Errorf(format string, args ...interface{})
		Fatalf(format string, args ...interface{})
		Panicf(format string, args ...interface{})
		Printf(format string, args ...interface{})
		WithFields(fields Fields) *Entry
		SetOutput(out io.Writer)
		SetLevel(level Level)
		SetFormatter(formatter Formatter)
	}

	Logger struct {
		output    io.Writer
		level     Level
		formatter Formatter
		sync.Mutex
	}

	Entry struct {
		Logger *Logger

		// Contains all the fields set by the user.
		Data Fields

		// Time at which the log entry was created
		Time time.Time

		// Level the log entry was logged at: Debug, Info, Warn, Error, Fatal or Panic
		Level Level

		// Message passed to Debug, Info, Warn, Error, Fatal or Panic
		Message string
	}

	Formatter interface {
		Format(*Entry) ([]byte, error)
	}

	AccessLogFmt struct {
	}

	ErrorLogFmt struct {
	}

	TextFormatter struct {
		TimestampFormat string
	}

	JSONFormatter struct {
		TimestampFormat string
	}
)

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel

	// Default Format
	ErrorLogTimestampFormat  = "Mon Jan 02 2006 15:04:05"
	AccessLogTimestampFormat = "02/Jan/2006:15:04:05 -0700"

	DefaultTimestampFormat = time.Stamp
)

// Convert the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}

	return "unknown"
}
