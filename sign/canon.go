// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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
