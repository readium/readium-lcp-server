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

package http

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/readium/readium-lcp-server/model"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

func FormToFields(deserializeTo reflect.Value, fromForm url.Values) error {
	for k, v := range fromForm {
		field := deserializeTo.Elem().FieldByName(k)
		val := v[0]
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val == "" {
				val = "0"
			}
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as int")
			} else {
				field.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if val == "" {
				val = "0"
			}
			uintVal, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as uint")
			} else {
				field.SetUint(uintVal)
			}
		case reflect.Bool:
			if val == "" {
				val = "false"
			}
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as boolean")
			} else {
				field.SetBool(boolVal)
			}
		case reflect.Float32:
			if val == "" {
				val = "0.0"
			}
			floatVal, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as 32-bit float")
			} else {
				field.SetFloat(floatVal)
			}
		case reflect.Float64:
			if val == "" {
				val = "0.0"
			}
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as 64-bit float")
			} else {
				field.SetFloat(floatVal)
			}
		case reflect.String:
			field.SetString(val)
		case reflect.Ptr:
			fv := reflect.New(field.Type())
			if iNull, ok := fv.Elem().Interface().(*model.NullTime); ok {
				iNull = model.Now()
				err := iNull.UnmarshalText([]byte(val))
				if err == nil {
					field.Set(reflect.ValueOf(iNull))
				} else {
					panic("Could not unmarshal text on model.NullTime :" + err.Error())
				}
			}
		}
	}
	return nil
}

/**
Link: <https://api.github.com/user/repos?page=3&per_page=100>; rel="next",  <https://api.github.com/user/repos?page=50&per_page=100>; rel="last"
next	The link relation for the immediate next page of results.
last	The link relation for the last page of results.
first	The link relation for the first page of results.
prev	The link relation for the immediate previous page of results.
*/
func ReadPagination(pg, perPg string, totalRecords int64) (int64, int64, error) {
	if pg == "" {
		pg = "0"
	}
	page, err := strconv.ParseInt(pg, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	if page < 0 {
		return 0, 0, errors.New("page must be positive integer")
	}
	if page > 0 { // starting at 0 in code, but user interface starting at 1
		page--
	}
	if perPg == "" {
		perPg = "0"
	}
	perPage, err := strconv.ParseInt(perPg, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	if perPage == 0 {
		perPage = 30
	}

	if totalRecords < page*perPage {
		return 0, 0, errors.New("page outside known range")
	}

	return page, perPage, nil
}

func MakePaginationHeader(addr string, page, perPage, total int64) string {
	t := "<%s?page=%d&per_page=%d>; rel=\"%s\""
	result := fmt.Sprintf(t, addr, page+1, perPage, "next") // next page
	if page > 1 {
		result += ",\n" + fmt.Sprintf(t, addr, page-1, perPage, "prev") // prev page
		result += ",\n" + fmt.Sprintf(t, addr, 1, perPage, "first")     // first page
	}
	f := (total - total%perPage) / perPage
	result += ",\n" + fmt.Sprintf(t, addr, f, perPage, "last") // last page
	return result
}

// Slugify creates the slug for a given value
func Slugify(value string) string {
	value = strings.ToLower(value)
	var buffer bytes.Buffer
	lastCharacterWasInvalid := false

	for len(value) > 0 {
		c, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		if newCharacter, ok := replacementMap[c]; ok {
			buffer.WriteString(newCharacter)
			lastCharacterWasInvalid = false
			continue
		}

		if validCharacter(c) {
			buffer.WriteRune(c)
			lastCharacterWasInvalid = false
		} else if !lastCharacterWasInvalid {
			buffer.WriteRune('-')
			lastCharacterWasInvalid = true
		}
	}

	return strings.Trim(buffer.String(), string('-'))
}

func validCharacter(c rune) bool {

	if c >= 'a' && c <= 'z' {
		return true
	}

	if c >= '0' && c <= '9' {
		return true
	}

	return false
}

var replacementMap = map[rune]string{
	'&': "and",
	'@': "at",
	'©': "c",
	'®': "r",
	'Æ': "ae",
	'ß': "ss",
	'à': "a",
	'á': "a",
	'â': "a",
	'ä': "ae",
	'å': "a",
	'æ': "ae",
	'ç': "c",
	'è': "e",
	'é': "e",
	'ê': "e",
	'ë': "e",
	'ì': "i",
	'í': "i",
	'î': "i",
	'ï': "i",
	'ò': "o",
	'ó': "o",
	'ô': "o",
	'õ': "o",
	'ö': "oe",
	'ø': "o",
	'ù': "u",
	'ú': "u",
	'û': "u",
	'ü': "ue",
	'ý': "y",
	'þ': "p",
	'ÿ': "y",
	'ā': "a",
	'ă': "a",
	'Ą': "a",
	'ą': "a",
	'ć': "c",
	'ĉ': "c",
	'ċ': "c",
	'č': "c",
	'ď': "d",
	'đ': "d",
	'ē': "e",
	'ĕ': "e",
	'ė': "e",
	'ę': "e",
	'ě': "e",
	'ĝ': "g",
	'ğ': "g",
	'ġ': "g",
	'ģ': "g",
	'ĥ': "h",
	'ħ': "h",
	'ĩ': "i",
	'ī': "i",
	'ĭ': "i",
	'į': "i",
	'ı': "i",
	'ĳ': "ij",
	'ĵ': "j",
	'ķ': "k",
	'ĸ': "k",
	'Ĺ': "l",
	'ĺ': "l",
	'ļ': "l",
	'ľ': "l",
	'ŀ': "l",
	'ł': "l",
	'ń': "n",
	'ņ': "n",
	'ň': "n",
	'ŉ': "n",
	'ŋ': "n",
	'ō': "o",
	'ŏ': "o",
	'ő': "o",
	'Œ': "oe",
	'œ': "oe",
	'ŕ': "r",
	'ŗ': "r",
	'ř': "r",
	'ś': "s",
	'ŝ': "s",
	'ş': "s",
	'š': "s",
	'ţ': "t",
	'ť': "t",
	'ŧ': "t",
	'ũ': "u",
	'ū': "u",
	'ŭ': "u",
	'ů': "u",
	'ű': "u",
	'ų': "u",
	'ŵ': "w",
	'ŷ': "y",
	'ź': "z",
	'ż': "z",
	'ž': "z",
	'ſ': "z",
	'Ə': "e",
	'ƒ': "f",
	'Ơ': "o",
	'ơ': "o",
	'Ư': "u",
	'ư': "u",
	'ǎ': "a",
	'ǐ': "i",
	'ǒ': "o",
	'ǔ': "u",
	'ǖ': "u",
	'ǘ': "u",
	'ǚ': "u",
	'ǜ': "u",
	'ǻ': "a",
	'Ǽ': "ae",
	'ǽ': "ae",
	'Ǿ': "o",
	'ǿ': "o",
	'ə': "e",
	'Є': "e",
	'Б': "b",
	'Г': "g",
	'Д': "d",
	'Ж': "zh",
	'З': "z",
	'У': "u",
	'Ф': "f",
	'Х': "h",
	'Ц': "c",
	'Ч': "ch",
	'Ш': "sh",
	'Щ': "sch",
	'Ъ': "-",
	'Ы': "y",
	'Ь': "-",
	'Э': "je",
	'Ю': "ju",
	'Я': "ja",
	'а': "a",
	'б': "b",
	'в': "v",
	'г': "g",
	'д': "d",
	'е': "e",
	'ж': "zh",
	'з': "z",
	'и': "i",
	'й': "j",
	'к': "k",
	'л': "l",
	'м': "m",
	'н': "n",
	'о': "o",
	'п': "p",
	'р': "r",
	'с': "s",
	'т': "t",
	'у': "u",
	'ф': "f",
	'х': "h",
	'ц': "c",
	'ч': "ch",
	'ш': "sh",
	'щ': "sch",
	'ъ': "-",
	'ы': "y",
	'ь': "-",
	'э': "je",
	'ю': "ju",
	'я': "ja",
	'ё': "jo",
	'є': "e",
	'і': "i",
	'ї': "i",
	'Ґ': "g",
	'ґ': "g",
	'א': "a",
	'ב': "b",
	'ג': "g",
	'ד': "d",
	'ה': "h",
	'ו': "v",
	'ז': "z",
	'ח': "h",
	'ט': "t",
	'י': "i",
	'ך': "k",
	'כ': "k",
	'ל': "l",
	'ם': "m",
	'מ': "m",
	'ן': "n",
	'נ': "n",
	'ס': "s",
	'ע': "e",
	'ף': "p",
	'פ': "p",
	'ץ': "C",
	'צ': "c",
	'ק': "q",
	'ר': "r",
	'ש': "w",
	'ת': "t",
	'™': "tm",
	'ả': "a",
	'ã': "a",
	'ạ': "a",

	'ắ': "a",
	'ằ': "a",
	'ẳ': "a",
	'ẵ': "a",
	'ặ': "a",

	'ấ': "a",
	'ầ': "a",
	'ẩ': "a",
	'ẫ': "a",
	'ậ': "a",

	'ẻ': "e",
	'ẽ': "e",
	'ẹ': "e",
	'ế': "e",
	'ề': "e",
	'ể': "e",
	'ễ': "e",
	'ệ': "e",

	'ỉ': "i",
	'ị': "i",

	'ỏ': "o",
	'ọ': "o",
	'ố': "o",
	'ồ': "o",
	'ổ': "o",
	'ỗ': "o",
	'ộ': "o",
	'ớ': "o",
	'ờ': "o",
	'ở': "o",
	'ỡ': "o",
	'ợ': "o",

	'ủ': "u",
	'ụ': "u",
	'ứ': "u",
	'ừ': "u",
	'ử': "u",
	'ữ': "u",
	'ự': "u",

	'ỳ': "y",
	'ỷ': "y",
	'ỹ': "y",
	'ỵ': "y",
}
