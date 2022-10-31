// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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
var ErrNotFound = errors.New("informations not found")

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
	db               *sql.DB
	dbGetBestSellers *sql.Stmt
}

// GetDashboardInfos a publication for a given ID
func (dashManager DashboardManager) GetDashboardInfos() (Dashboard, error) {
	//
	var dash Dashboard

	row := dashManager.db.QueryRow("SELECT COUNT(*) FROM publication")
	row.Scan(&dash.PublicationCount)

	row = dashManager.db.QueryRow("SELECT COUNT(*) FROM user")
	row.Scan(&dash.UserCount)

	row = dashManager.db.QueryRow(`SELECT COUNT(*) FROM purchase WHERE type="BUY"`)
	row.Scan(&dash.BuyCount)

	row = dashManager.db.QueryRow(`SELECT COUNT(*) FROM purchase WHERE type="LOAN"`)
	row.Scan(&dash.LoanCount)

	return dash, nil
}

// GetDashboardBestSellers a publication for a given ID
func (dashManager DashboardManager) GetDashboardBestSellers() ([5]BestSeller, error) {
	rows, err := dashManager.dbGetBestSellers.Query()
	if err != nil {
		return [5]BestSeller{}, err
	}
	defer rows.Close()

	var bestSellers [5]BestSeller
	i := 0

	for rows.Next() {
		var newBestSeller BestSeller
		err := rows.Scan(&newBestSeller.Title, &newBestSeller.Count)
		bestSellers[i] = newBestSeller
		i++
		if err != nil {
			return bestSellers, err
		}
	}
	return bestSellers, err
}

// Init publication manager
func Init(db *sql.DB) (i WebDashboard, err error) {

	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	var dbGetBestSellers *sql.Stmt

	if driver == "sqlserver" {
		dbGetBestSellers, err = db.Prepare(
			`SELECT TOP 5 pub.title, count(pub.id)
				FROM purchase pur 
				JOIN publication pub 
				ON pur.publication_id = pub.id
				GROUP BY pub.id
				ORDER BY  Count(pur.id) DESC`)
	} else {
		dbGetBestSellers, err = db.Prepare(
			`SELECT pub.title, count(pub.id)
				FROM purchase pur 
				JOIN publication pub 
				ON pur.publication_id = pub.id
				 GROUP BY pub.id
				 ORDER BY  Count(pur.id) DESC LIMIT 5`)
	}

	i = DashboardManager{db, dbGetBestSellers}
	return
}
