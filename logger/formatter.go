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
	"encoding/json"
	"fmt"
)

var (
	// Make sure all formatters implement Interface
	_ Formatter = &AccessLogFmt{}
	_ Formatter = &ErrorLogFmt{}
	_ Formatter = &TextFormatter{}
	_ Formatter = &JSONFormatter{}
)

func (a *AccessLogFmt) Format(entry *Entry) ([]byte, error) {

	client, found := entry.Data["ip"]
	if !found {
		client = "0.0.0.0"
	}

	status, found := entry.Data["status"]
	if !found {
		status = 200
	}

	size, found := entry.Data["size"]
	if !found {
		size = 0
	}

	agent, found := entry.Data["agent"]
	if !found {
		agent = ""
	}

	return []byte(
		fmt.Sprintf("%s - - [%s] \"%s\" %d %d \"-\" \"%s\"\n",
			client,
			entry.Time.Format(AccessLogTimestampFormat),
			entry.Message,
			status,
			size,
			agent,
		),
	), nil
}

func (e *ErrorLogFmt) Format(entry *Entry) ([]byte, error) {

	client, found := entry.Data["ip"]
	if !found {
		client = "0.0.0.0"
	}

	return []byte(
		fmt.Sprintf("[%s] [%s] [client %s] %s\n",
			entry.Time.Format(ErrorLogTimestampFormat),
			entry.Level.String(),
			client,
			entry.Message,
		),
	), nil
}

func (t *TextFormatter) Format(entry *Entry) ([]byte, error) {

	timestampFormat := t.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultTimestampFormat
	}

	return []byte(
		fmt.Sprintf("%s [%s] %s\n",
			entry.Time.Format(timestampFormat),
			entry.Level.String(),
			entry.Message,
		),
	), nil
}

func (j *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(Fields, 3)

	timestampFormat := j.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultTimestampFormat
	}

	data["time"] = entry.Time.Format(timestampFormat)
	data["msg"] = entry.Message
	data["level"] = entry.Level.String()

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}
