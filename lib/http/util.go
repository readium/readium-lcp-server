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
	"errors"
	"fmt"
	"strconv"
)

/**
Link: <https://api.github.com/user/repos?page=3&per_page=100>; rel="next",  <https://api.github.com/user/repos?page=50&per_page=100>; rel="last"
next	The link relation for the immediate next page of results.
last	The link relation for the last page of results.
first	The link relation for the first page of results.
prev	The link relation for the immediate previous page of results.
*/
func ReadPagination(pg, perPg string, totalRecords int64) (int64, int64, error) {
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
