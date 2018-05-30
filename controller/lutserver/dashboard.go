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

package lutserver

import (
	"github.com/jinzhu/gorm"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/model"
)

// GetDashboardInfos searches a publication by its uuid
func GetDashboardInfos(server http.IServer) (*model.Dashboard, error) {
	if pub, err := server.Store().Dashboard().GetDashboardInfos(); err == nil {
		return pub, nil
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
}

// GetDashboardBestSellers gets the dashboard bestsellers
//
func GetDashboardBestSellers(server http.IServer) ([]model.BestSeller, error) {
	if pub, err := server.Store().Dashboard().GetDashboardBestSellers(); err == nil {
		return pub, nil
	} else {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusNotFound}
		default:
			return nil, http.Problem{Detail: err.Error(), Status: http.StatusInternalServerError}
		}
	}
}
