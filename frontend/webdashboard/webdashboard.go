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

package webdashboard

import (
	"database/sql"
	"errors"

	"github.com/readium/readium-lcp-server/config"
)

// Publication status
const (
	StatusDraft      string = "draft"
	StatusEncrypting string = "encrypting"
	StatusError      string = "error"
	StatusOk         string = "ok"
)

// ErrNotFound error trown when publication is not found
var ErrNotFound = errors.New("Informations not found")

// WebDashboard interface for publication db interaction
type WebDashboard interface {
	GetDashboardInfos() (Dashboard, error)
	GetDashboardBestSellers() ([5]BestSeller, error)
}

// Dashboard struct defines a publication
type Dashboard struct {
	PublicationCount int64 `json:"publicationCount"`
	UserCount        int64 `json:"userCount"`
	BuyCount         int64 `json:"buyCount"`
	LoanCount        int64 `json:"loanCount"`
}

// BestSeller struct defines a best seller
type BestSeller struct {
	Title string `json:"title"`
	Count int64  `json:"count"`
}

// DashboardManager helper
type DashboardManager struct {
	config config.Configuration
	db     *sql.DB
}

// GetDashboardInfos a publication for a given ID
func (dashManager DashboardManager) GetDashboardInfos() (Dashboard, error) {
	//
	var dash Dashboard

	dbGet, err := dashManager.db.Prepare("SELECT COUNT(*) FROM publication")
	if err != nil {
		return Dashboard{}, err
	}
	defer dbGet.Close()

	records, err := dbGet.Query()
	if records.Next() {
		err = records.Scan(&dash.PublicationCount)
		records.Close()
	}
	//
	dbGet, err = dashManager.db.Prepare("SELECT COUNT(*) FROM user")
	if err != nil {
		return Dashboard{}, err
	}
	defer dbGet.Close()

	records, err = dbGet.Query()
	if records.Next() {
		err = records.Scan(&dash.UserCount)
		records.Close()
	}
	//
	dbGet, err = dashManager.db.Prepare(`SELECT COUNT(*) FROM purchase WHERE type="BUY"`)
	if err != nil {
		return Dashboard{}, err
	}
	defer dbGet.Close()

	records, err = dbGet.Query()
	if records.Next() {
		err = records.Scan(&dash.BuyCount)
		records.Close()
	}
	//
	dbGet, err = dashManager.db.Prepare(`SELECT COUNT(*) FROM purchase WHERE type="LOAN"`)
	if err != nil {
		return Dashboard{}, err
	}
	defer dbGet.Close()

	records, err = dbGet.Query()
	if records.Next() {
		err = records.Scan(&dash.LoanCount)
		records.Close()
	}

	return dash, nil
}

// GetDashboardBestSellers a publication for a given ID
func (dashManager DashboardManager) GetDashboardBestSellers() ([5]BestSeller, error) {
	dbList, err := dashManager.db.Prepare(
		`SELECT pub.title, count(pub.id)
  		FROM purchase pur JOIN publication pub 
    	ON pur.publication_id = pub.id
 		GROUP BY pub.id
 		ORDER BY  Count(pur.id) DESC limit 5`)
	if err != nil {
		return [5]BestSeller{}, err
	}
	defer dbList.Close()
	records, err := dbList.Query()
	if err != nil {
		return [5]BestSeller{}, err
	}

	var bestSellers [5]BestSeller
	i := 0
	for records.Next() {
		var newBestSeller BestSeller
		err := records.Scan(&newBestSeller.Title, &newBestSeller.Count)
		bestSellers[i] = newBestSeller
		i++
		if err != nil {
			return bestSellers, err
		}

	}
	return bestSellers, err
}

// Init publication manager
func Init(config config.Configuration, db *sql.DB) (i WebDashboard, err error) {
	i = DashboardManager{config, db}
	return
}
