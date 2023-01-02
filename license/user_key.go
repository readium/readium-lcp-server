// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package license

import (
	"log"
	"regexp"
)

// GenerateUserKey function prepares the user key
func GenerateUserKey(key UserKey, profile string) []byte {
	if match, _ := regexp.MatchString("^http://readium.org/lcp/profile-2.[0-9]$", profile); !match && profile != "http://readium.org/lcp/basic-profile" && profile != "http://readium.org/lcp/profile-1.0" {
		log.Printf("Incompatible LCP profile, got %s", profile)
		return nil
	}
	return key.Value
}
