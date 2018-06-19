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

package helpers

import (
	"fmt"
	"github.com/readium/readium-lcp-server/model"
	got "html/template"
	"reflect"
	"strings"
	"time"
)

// ARRAYS

// Array takes a set of interface pointers as variadic args, and returns a single array
func Array(args ...interface{}) []interface{} {
	return []interface{}{args}
}

// CommaSeparatedArray returns the values as a comma separated string
func CommaSeparatedArray(args []string) string {
	result := ""
	for _, v := range args {
		if len(result) > 0 {
			result = fmt.Sprintf("%s,%s", result, v)
		} else {
			result = v
		}

	}
	return result
}

// MAPS

// Empty returns an empty map[string]interface{} for use as a context
func Empty() map[string]interface{} {
	return map[string]interface{}{}
}

// Map sets a map key and return the map
func Map(m map[string]interface{}, k string, v interface{}) map[string]interface{} {
	m[k] = v
	return m
}

// Set a map key and return an empty string
func Set(m map[string]interface{}, k string, v interface{}) string {
	m[k] = v
	return "" // Render nothing, we want no side effects
}

// SetIf sets a map key if the given condition is true
func SetIf(m map[string]interface{}, k string, v interface{}, t bool) string {
	if t {
		m[k] = v
	} else {
		m[k] = ""
	}
	return "" // Render nothing, we want no side effects
}

// Append all args to an array, and return that array
func Append(m []interface{}, args ...interface{}) []interface{} {
	for _, v := range args {
		m = append(m, v)
	}
	return m
}

// CreateMap - given a set of interface pointers as variadic args, generate and return a map to the values
// This is currently unused as we just use simpler Map add above to add to context
func CreateMap(args ...interface{}) map[string]interface{} {
	m := make(map[string]interface{}, 0)
	key := ""
	for _, v := range args {
		if len(key) == 0 {
			key = string(v.(string))
		} else {
			m[key] = v
		}
	}
	return m
}

// Contains returns true if this array of ints contains the given int
func Contains(list []int64, item int64) bool {
	for _, b := range list {
		if b == item {
			return true
		}
	}
	return false
}

// Blank returns true if a string is empty
func Blank(s string) bool {
	return len(s) == 0
}

// Exists returns true if this string has a length greater than 0
func Exists(s string) bool {
	return len(s) > 0
}

// Time returns a formatted time string given a time and optional format
func Time(time time.Time, formats ...string) got.HTML {
	layout := "Jan 2, 2006 15:04"
	if len(formats) > 0 {
		layout = formats[0]
	}
	value := fmt.Sprintf(time.Format(layout))
	return got.HTML(Escape(value))
}

// Date returns a formatted date string given a time and optional format
// Date format layouts are for the date 2006-01-02
func Date(t time.Time, formats ...string) got.HTML {

	//layout := "2006-01-02" // Jan 2, 2006
	layout := "Jan 2, 2006"
	if len(formats) > 0 {
		layout = formats[0]
	}
	value := fmt.Sprintf(t.Format(layout))
	return got.HTML(Escape(value))
}

// UTCDate returns a formatted date string in 2006-01-02
func UTCDate(t interface{}) got.HTML {
	switch tim := t.(type) {
	case time.Time:
		return Date(tim.UTC(), "2006-01-02")
	case model.NullTime:
		if tim.Valid {
			return Date(tim.Time.UTC(), "2006-01-02")
		}
		return "-"
	case *model.NullTime:
		if tim == nil {
			return "-"
		}
		if tim.Valid {
			return Date(tim.Time.UTC(), "2006-01-02")
		}
		return "-"
	default:
		fmt.Printf("And interface is %#v\n", tim)
		panic("Bad usage of utcdate (requires time.Time or NullTime)")
	}
}

// UTCTime returns a formatted date string in 2006-01-02
func UTCTime(t time.Time) got.HTML {
	return Time(t.UTC(), "2006-01-02T15:04:00:00.000Z")
}

// UTCNow returns a formatted date string in 2006-01-02
func UTCNow() got.HTML {
	return Date(time.Now(), "2006-01-02")
}

// Truncate text to a given length
func Truncate(s string, l int64) string {
	return s
}

// CSV escape (replace , with ,,)
func CSV(s got.HTML) string {
	return strings.Replace(string(s), ",", ",,", -1)
}

func Last(x int, a interface{}) bool {
	return x == reflect.ValueOf(a).Len()-1
}

func IsEmpty(a interface{}) bool {
	return reflect.ValueOf(a).Len() == 0
}
func NotEmpty(a interface{}) bool {
	if a == nil {
		return false
	}
	return reflect.ValueOf(a).Len() > 0
}
func Defined(a interface{}) bool {
	return a != nil
}
