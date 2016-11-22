// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. 

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

	TYPE_REGISTER = "register"
	TYPE_RETURN   = "return"
	TYPE_RENEW    = "renew"
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
	1: TYPE_REGISTER,
	2: TYPE_RETURN,
	3: TYPE_RENEW,
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
