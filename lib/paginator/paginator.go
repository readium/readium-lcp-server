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

package paginator

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
)

// Paginator within the state of a http request.
type Paginator struct {
	Request     *http.Request
	PerPageNums int64
	MaxPages    int64

	nums      int64
	pageRange []int64
	pageNums  int64
	page      int64
}

// PageNums Returns the total number of pages.
func (p *Paginator) PageNums() int64 {
	if p.pageNums != 0 {
		return p.pageNums
	}
	pageNums := math.Ceil(float64(p.nums) / float64(p.PerPageNums))
	if p.MaxPages > 0 {
		pageNums = math.Min(pageNums, float64(p.MaxPages))
	}
	p.pageNums = int64(pageNums)
	return p.pageNums
}

// Nums Returns the total number of items (e.g. from doing SQL count).
func (p *Paginator) Nums() int64 {
	return p.nums
}

// SetNums Sets the total number of items.
func (p *Paginator) SetNums(nums int64) {
	p.nums = nums
}

// Page Returns the current page.
func (p *Paginator) Page() int64 {
	if p.page != 0 {
		return p.page
	}
	if p.Request.Form == nil {
		p.Request.ParseForm()
	}
	v, _ := strconv.Atoi(p.Request.Form.Get("p"))
	p.page = int64(v)
	if p.page > p.PageNums() {
		p.page = p.PageNums()
	}
	if p.page <= 0 {
		p.page = 1
	}
	return p.page
}

// Pages Returns a list of all pages.
//
// Usage (in a view template):
//
//  {{range $index, $page := .paginator.Pages}}
//    <li{{if $.paginator.IsActive .}} class="active"{{end}}>
//      <a href="{{$.paginator.PageLink $page}}">{{$page}}</a>
//    </li>
//  {{end}}
func (p *Paginator) Pages() []int64 {
	if p.pageRange == nil && p.nums > 0 {
		var pages []int64
		pageNums := p.PageNums()
		page := p.Page()
		switch {
		case page >= pageNums-4 && pageNums > 9:
			start := pageNums - 9 + 1
			pages = make([]int64, 9)
			for i := range pages {
				pages[i] = start + i
			}
		case page >= 5 && pageNums > 9:
			start := page - 5 + 1
			pages = make([]int64, int(math.Min(9, float64(page+4+1))))
			for i := range pages {
				pages[i] = start + i
			}
		default:
			pages = make([]int64, int(math.Min(9, float64(pageNums))))
			for i := range pages {
				pages[i] = i + 1
			}
		}
		p.pageRange = pages
	}
	return p.pageRange
}

// PageLink Returns URL for a given page index.

func (p *Paginator) PageLink(page int64) string {
	link, _ := url.ParseRequestURI(p.Request.URL.String())
	values := link.Query()
	if page == 1 {
		values.Del("p")
	} else {
		values.Set("p", strconv.Itoa(int(page)))
	}
	link.RawQuery = values.Encode()
	return link.String()
}

// PageLinkPrev Returns URL to the previous page.
func (p *Paginator) PageLinkPrev() (link string) {
	if p.HasPrev() {
		link = p.PageLink(p.Page() - 1)
	}
	return
}

// PageLinkNext Returns URL to the next page.
func (p *Paginator) PageLinkNext() (link string) {
	if p.HasNext() {
		link = p.PageLink(p.Page() + 1)
	}
	return
}

// PageLinkFirst Returns URL to the first page.
func (p *Paginator) PageLinkFirst() (link string) {
	return p.PageLink(1)
}

// PageLinkLast Returns URL to the last page.
func (p *Paginator) PageLinkLast() (link string) {
	return p.PageLink(p.PageNums())
}

// HasPrev Returns true if the current page has a predecessor.
func (p *Paginator) HasPrev() bool {
	return p.Page() > 1
}

// HasNext Returns true if the current page has a successor.
func (p *Paginator) HasNext() bool {
	return p.Page() < p.PageNums()
}

// IsActive Returns true if the given page index points to the current page.
func (p *Paginator) IsActive(page int64) bool {
	return p.Page() == page
}

// Offset Returns the current offset.
func (p *Paginator) Offset() int64 {
	return (p.Page() - 1) * p.PerPageNums
}

// HasPages Returns true if there is more than one page.
func (p *Paginator) HasPages() bool {
	return p.PageNums() > 1
}

//stringer implementation
func (p *Paginator) String() string {
	return fmt.Sprintf("nums:%d page:%d max:%d perpage:%d", p.nums, p.page, p.MaxPages, p.PerPageNums)
}

// NewPaginator Instantiates a paginator struct for the current http request.
func NewPaginator(req *http.Request, per int64, nums int64) *Paginator {
	p := Paginator{}
	p.Request = req
	if per <= 0 {
		per = 30
	}
	p.PerPageNums = per
	p.SetNums(nums)
	return &p
}

// SetPaginator Instantiates a Paginator and assigns it to context.Input.Data["paginator"].
func SetPaginator(request *http.Request, per int64, nums int64) (paginator *Paginator) {
	return NewPaginator(request, per, nums)
}
