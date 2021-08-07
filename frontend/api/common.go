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

package staticapi

import (
	"net/http"
	"strconv"

	"github.com/endigo/readium-lcp-server/api"
	"github.com/endigo/readium-lcp-server/frontend/webdashboard"
	"github.com/endigo/readium-lcp-server/frontend/weblicense"
	"github.com/endigo/readium-lcp-server/frontend/webpublication"
	"github.com/endigo/readium-lcp-server/frontend/webpurchase"
	"github.com/endigo/readium-lcp-server/frontend/webrepository"
	"github.com/endigo/readium-lcp-server/frontend/webuser"
)

//IServer defines methods for db interaction
type IServer interface {
	RepositoryAPI() webrepository.WebRepository
	PublicationAPI() webpublication.WebPublication
	UserAPI() webuser.WebUser
	PurchaseAPI() webpurchase.WebPurchase
	DashboardAPI() webdashboard.WebDashboard
	LicenseAPI() weblicense.WebLicense
}

// Pagination used to paginate listing
type Pagination struct {
	Page    int
	PerPage int
}

// ExtractPaginationFromRequest extract from http.Request pagination information
func ExtractPaginationFromRequest(r *http.Request) (Pagination, error) {
	var err error
	var page int64    // default: page 1
	var perPage int64 // default: 30 items per page
	pagination := Pagination{}

	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			return pagination, err
		}
	} else {
		page = 1
	}

	if r.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			return pagination, err
		}
	} else {
		perPage = 30
	}

	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}

	if page < 0 {
		return pagination, err
	}

	pagination.Page = int(page)
	pagination.PerPage = int(perPage)
	return pagination, err
}

// PrepareListHeaderResponse set several http headers
// sets previous and next link headers
func PrepareListHeaderResponse(resourceCount int, resourceLink string, pagination Pagination, w http.ResponseWriter) {
	if resourceCount > 0 {
		nextPage := strconv.Itoa(int(pagination.Page) + 1)
		w.Header().Set("Link", "<"+resourceLink+"?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if pagination.Page > 1 {
		previousPage := strconv.Itoa(int(pagination.Page) - 1)
		w.Header().Set("Link", "<"+resourceLink+"/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	w.Header().Set("Content-Type", api.ContentType_JSON)
}
