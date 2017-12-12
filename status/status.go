// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package status

import (
	"strconv"
	"strings"
)

// List of status values
const (
	STATUS_READY     = "ready"
	STATUS_ACTIVE    = "active"
	STATUS_REVOKED   = "revoked"
	STATUS_RETURNED  = "returned"
	STATUS_CANCELLED = "cancelled"
	STATUS_EXPIRED   = "expired"
	EVENT_RENEWED    = "renewed"
)

// StatusValues defines event types, used in events logged in license status documents
var StatusValues = map[int]string{
	0: STATUS_READY,
	1: STATUS_ACTIVE,
	2: STATUS_REVOKED,
	3: STATUS_RETURNED,
	4: STATUS_CANCELLED,
	5: STATUS_EXPIRED,
}

// EventTypes defines additional event types.
// It reuses all status values and adds one for renewed licenses.
var EventTypes = map[int]string{
	1: STATUS_ACTIVE,
	2: STATUS_REVOKED,
	3: STATUS_RETURNED,
	4: STATUS_CANCELLED,
	5: STATUS_EXPIRED,
	6: EVENT_RENEWED,
}

// GetStatus translates status number to status string
func GetStatus(statusDB int64, status *string) {
	resultStr := reverse(strconv.FormatInt(statusDB, 2))

	if count := strings.Count(resultStr, "1"); count == 1 {
		index := strings.Index(resultStr, "1")

		if len(StatusValues) >= index+1 {
			*status = StatusValues[index]
		}
	}
}

// SetStatus translates status string to status number
func SetStatus(status string) (int64, error) {
	reg := make([]string, len(StatusValues))

	for key := range StatusValues {
		if StatusValues[key] == status {
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
