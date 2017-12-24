// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import "crypto/sha256"

// GenerateUserKey function prepares the user key
func GenerateUserKey(key UserKey) []byte {
	var userKey []byte
	userKey = key.Value
	return userKey
}
