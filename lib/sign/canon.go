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

package sign

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

func Canon(in interface{}) ([]byte, error) {
	// the easiest way to canonicalize is to marshal it and reify it as a map
	// which will sort stuff correctly
	b, err1 := json.Marshal(in)
	if err1 != nil {
		return b, err1
	}

	var jsonObj interface{} // map[string]interface{} ==> auto sorting

	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()
	for {
		if err2 := dec.Decode(&jsonObj); err2 == io.EOF {
			break
		} else if err2 != nil {
			return nil, err2
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err := enc.Encode(jsonObj)
	if err != nil {
		return nil, err
	}
	// remove the trailing newline, added by encode
	return bytes.TrimRight(buf.Bytes(), "\n"), nil

}
