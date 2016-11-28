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

package webpurchase

import (
	"database/sql"
	"errors"
	"time"

	"github.com/readium/readium-lcp-server/static/webuser"
)

var ErrNotFound = errors.New("User not found")

type WebPurchase interface {
	Get(id int64) (Purchase, error)
	Add(p Purchase) error
	Update(p Purchase) error
}

//User struct defines a user in json and database
type Purchase struct {
	User            webuser.User `json:"user"`
	PurchaseID      int          `json:"purchaseID"`
	Resource        string       `json:"resource"`
	Label           string       `json:"label"`
	TransactionDate time.Time    `json:"transactionDate"`
	PartialLicense  string       `json:"partialLicense"`
}

type dbPurchase struct {
	db          *sql.DB
	get         *sql.Stmt
	add         *sql.Stmt
	listForUser *sql.Stmt
}

func (purchase dbPurchase) Get(id int64) (Purchase, error) {
	records, err := purchase.get.Query(id)
	defer records.Close()
	if records.Next() {
		var p Purchase
		p.User = webuser.User{}
		// purchase_id, user_id, resource, label, transaction_date, user_id, alias, email, password
		err = records.Scan(&p.PurchaseID, &p.Resource, &p.Label, &p.TransactionDate, &p.User.UserID, &p.User.Alias, &p.User.Email, &p.User.Password)
		return p, err
	}

	return Purchase{}, ErrNotFound
}

func (user dbPurchase) Add(p Purchase) error {
	add, err := user.db.Prepare("INSERT INTO purchase ( purchase_id, user_id, resource, label, transaction_date, partialLicense) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(p.PurchaseID, p.User.UserID, p.Label, p.PartialLicense, p.PartialLicense)
	return err
}

func (user dbPurchase) Update(changedPurchase Purchase) error {
	add, err := user.db.Prepare("UPDATE purchase SET user_id=?, resource=?, label=?, transaction_date=?, partialLicense=? WHERE purchase_id=?")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(changedPurchase.User.UserID, changedPurchase.Resource, changedPurchase.Label, changedPurchase.TransactionDate, changedPurchase.PartialLicense, changedPurchase.PurchaseID)
	return err
}

func Open(db *sql.DB) (i WebPurchase, err error) {
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS purchase (
	purchase_id varchar(255) PRIMARY KEY, 
	user_id int NOT NULL, 
	resource varchar(64) NOT NULL, 
	label varchar(64) NOT NULL, 
    transaction_date datetime,
    partialLicense varchar(8192)
	constraint pk_purchase  primary key(id),
    constraint fk_purchase_user foreign key (user_id) references user(id)
	)`)
	if err != nil {
		return
	}
	get, err := db.Prepare(`SELECT 
      purchase_id, resource, label, transaction_date, user_id, alias, email, password 
    FROM purchase p 
    inner join user u on (p.user_id=u.id) 
    WHERE purchase_id = ? LIMIT 1`)
	if err != nil {
		return
	}
	i = dbPurchase{db, get, nil, nil}
	return
}
