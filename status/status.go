// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package status

import (
	"strconv"
	"strings"
)

const (
	STATUS_READY     = "ready"
	STATUS_ACTIVE    = "active"
	STATUS_REVOKED   = "revoked"
	STATUS_RETURNED  = "returned"
	STATUS_CANCELLED = "cancelled"
	STATUS_EXPIRED   = "expired"

	EVENT_REGISTERED = "registered"
	EVENT_RETURNED   = "returned"
	EVENT_RENEWED    = "renewed"
	EVENT_CANCELLED  = "cancelled"
	EVENT_REVOKED    = "revoked"
)

var statuses = map[int]string{
	0: STATUS_READY,
	1: STATUS_ACTIVE,
	2: STATUS_REVOKED,
	3: STATUS_RETURNED,
	4: STATUS_CANCELLED,
	5: STATUS_EXPIRED,
}

var Types = map[int]string{
	1: EVENT_REGISTERED,
	2: EVENT_RETURNED,
	3: EVENT_RENEWED,
	4: EVENT_CANCELLED,
	5: EVENT_REVOKED,
}

//GetStatus translate status number to status string
func GetStatus(statusDB int64, status *string) {
	resultStr := reverse(strconv.FormatInt(statusDB, 2))

	if count := strings.Count(resultStr, "1"); count == 1 {
		index := strings.Index(resultStr, "1")

		if len(statuses) >= index+1 {
			*status = statuses[index]
		}
	}
}

//SetStatus translate status string to status number
func SetStatus(status string) (int64, error) {
	reg := make([]string, len(statuses))

	for key := range statuses {
		if statuses[key] == status {
			reg[key] = "1"
		} else {
			reg[key] = "0"
		}
	}

	resultStr := reverse(strings.Join(reg[:], ""))

	statusDB, err := strconv.ParseInt(resultStr, 2, 64)
	return statusDB, err
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
