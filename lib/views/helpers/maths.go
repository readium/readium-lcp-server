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
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// PRICES

// FIXME - move to currency type with concrete implementations per currency, as it'd be neater than funcs with multiple options.  currency.GBP.PriceToCents something like that?

// PriceToCentsString returns a price in cents as a string for use in params
func PriceToCentsString(p string) string {
	if p == "" {
		return "0" // Return 0 for blank price
	}

	return fmt.Sprintf("%d", PriceToCents(p))
}

func FormatFloatNoDecimals(floa float64) string {

	src := strconv.FormatFloat(floa, 'f', 0, 64)
	lastIndex := strings.Index(src, ".") - 1

	if lastIndex < 0 {
		lastIndex = len(src) - 1
	}

	var buffer []byte
	var strBuffer bytes.Buffer

	j := 0
	for i := lastIndex; i >= 0; i-- {
		j++
		buffer = append(buffer, src[i])

		if j == 3 && i > 0 && !(i == 1 && src[0] == '-') {
			buffer = append(buffer, '.')
			j = 0
		}
	}

	for i := len(buffer) - 1; i >= 0; i-- {
		strBuffer.WriteByte(buffer[i])
	}
	result := strBuffer.String()

	//if thousand != "," {
	//	result = strings.Replace(result, ",", thousand, -1)
	//}

	extra := src[lastIndex+1:]
	//if decimal != "." {
	//extra = strings.Replace(extra, ".", decimal, 1)
	//}

	return result + extra
}

// PriceToCents converts a price string in human friendly notation (£45 or £34.40) to a price in pence as an int64
func PriceToCents(p string) int {
	price := strings.Replace(p, "£", "", -1)
	price = strings.Replace(price, ",", "", -1) // assumed to be in thousands
	price = strings.Replace(price, " ", "", -1)
	var pennies int
	var err error
	if strings.Contains(price, ".") {
		// Split the string on . and rejoin with padded pennies
		parts := strings.Split(price, ".")
		if len(parts[1]) == 0 {
			parts[1] = "00"
		} else if len(parts[1]) == 1 {
			parts[1] = parts[1] + "0"
		}
		price = parts[0] + parts[1]
		pennies, err = strconv.Atoi(price)
	} else {
		pennies, err = strconv.Atoi(price)
		pennies = pennies * 100
	}
	if err != nil {
		fmt.Printf("Error converting price %s", price)
		pennies = 0
	}
	return pennies
}

// CentsToPrice converts a price in pence to a human friendly price including currency unit
// At present it assumes the currency is pounds, it should instead take an optional param for currency
// or not include it at all
func CentsToPrice(p int64) string {
	price := fmt.Sprintf("£%.2f", float64(p)/100.0)
	return strings.TrimSuffix(price, ".00") // remove zero pence at end if we have it
}

// CentsToPriceShort converts a price in pence to a human friendly price abreviated (no pence)
func CentsToPriceShort(p int64) string {
	// If greater than £1000 return trimmed price
	if p > 99900 {
		return fmt.Sprintf("£%.1fk", float64(p)/100000.0)
	}
	return CentsToPrice(p)
}

// CentsToBase converts cents to the base currency unit, preserving cent display, with no currency
func CentsToBase(p int64) string {
	return fmt.Sprintf("%.2f", float64(p)/100.0)
}

// Mod returns a modulo b
func Mod(a int, b int) int {
	return a % b
}

// Add returns a + b
func Add(a int, b int) int {
	return a + b
}

// Subtract returns a - b
func Subtract(a int, b int) int {
	return a - b
}

// Odd returns true if a is odd
func Odd(a int) bool {
	return a%2 == 0
}
