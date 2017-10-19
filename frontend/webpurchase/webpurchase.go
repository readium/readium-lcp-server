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
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webuser"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/license_statuses"
	"github.com/satori/go.uuid"
)

//ErrNotFound Error is thrown when a purchase is not found
var ErrNotFound = errors.New("Purchase not found")

//ErrNoChange is thrown when an update action does not change any rows (not found)
var ErrNoChange = errors.New("No lines were updated")

const purchaseManagerQuery = `SELECT
p.id, p.uuid,
p.type, p.transaction_date,
p.license_uuid,
p.start_date, p.end_date, p.status,
u.id, u.uuid, u.name, u.email, u.password, u.hint,
pu.id, pu.uuid, pu.title, pu.status
FROM purchase p
left join user u on (p.user_id=u.id)
left join publication pu on (p.publication_id=pu.id)`

//WebPurchase defines possible interactions with DB
type WebPurchase interface {
	Get(id int64) (Purchase, error)
	GenerateLicense(purchase Purchase) (license.License, error)
	GetPartialLicense(purchase Purchase) (license.License, error)
	GetLicenseStatusDocument(purchase Purchase) (licensestatuses.LicenseStatus, error)
	GetByLicenseID(licenseID string) (Purchase, error)
	List(page int, pageNum int) func() (Purchase, error)
	ListByUser(userID int64, page int, pageNum int) func() (Purchase, error)
	Add(p Purchase) error
	Update(p Purchase) error
}

// Purchase status
const (
	StatusToBeRenewed  string = "to-be-renewed"
	StatusToBeReturned string = "to-be-returned"
	StatusError        string = "error"
	StatusOk           string = "ok"
)

// Enumeration of PurchaseType
const (
	BUY  string = "BUY"
	LOAN string = "LOAN"
)

//Purchase struct defines a user in json and database
//PurchaseType: BUY or LOAN
type Purchase struct {
	ID              int64                      `json:"id, omitempty"`
	UUID            string                     `json:"uuid"`
	Publication     webpublication.Publication `json:"publication"`
	User            webuser.User               `json:"user"`
	LicenseUUID     *string                    `json:"licenseUuid,omitempty"`
	Type            string                     `json:"type"`
	TransactionDate time.Time                  `json:"transactionDate, omitempty"`
	StartDate       *time.Time                 `json:"startDate, omitempty"`
	EndDate         *time.Time                 `json:"endDate, omitempty"`
	Status          string                     `json:"status"`
	MaxEndDate      *time.Time                 `json:"maxEndDate, omitempty"`
}

type purchaseManager struct {
	config config.Configuration
	db     *sql.DB
}

func convertRecordsToPurchases(records *sql.Rows) func() (Purchase, error) {
	return func() (Purchase, error) {
		var err error
		var purchase Purchase
		if records.Next() {
			purchase, err = convertRecordToPurchase(records)
			if err != nil {
				return purchase, err
			}
		} else {
			records.Close()
			err = ErrNotFound
		}

		return purchase, err
	}
}

func convertRecordToPurchase(records *sql.Rows) (Purchase, error) {
	purchase := Purchase{}
	user := webuser.User{}
	pub := webpublication.Publication{}

	err := records.Scan(
		&purchase.ID,
		&purchase.UUID,
		&purchase.Type,
		&purchase.TransactionDate,
		&purchase.LicenseUUID,
		&purchase.StartDate,
		&purchase.EndDate,
		&purchase.Status,
		&user.ID,
		&user.UUID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Hint,
		&pub.ID,
		&pub.UUID,
		&pub.Title,
		&pub.Status)

	if err != nil {
		return Purchase{}, err
	}

	// Load relations
	purchase.User = user
	purchase.Publication = pub
	return purchase, err
}

func (pManager purchaseManager) Get(id int64) (Purchase, error) {
	dbGetQuery := purchaseManagerQuery + ` WHERE p.id = ? LIMIT 1`
	dbGet, err := pManager.db.Prepare(dbGetQuery)
	if err != nil {
		return Purchase{}, err
	}
	defer dbGet.Close()

	records, err := dbGet.Query(id)
	defer records.Close()

	if records.Next() {
		purchase, err := convertRecordToPurchase(records)

		if err != nil {
			return Purchase{}, err
		}

		if purchase.LicenseUUID != nil {
			// Query LSD to retrieve max end date (PotentialRights.End)
			statusDocument, err := pManager.GetLicenseStatusDocument(purchase)

			if err != nil {
				return Purchase{}, err
			}

			if statusDocument.PotentialRights != nil && statusDocument.PotentialRights.End != nil && !(*statusDocument.PotentialRights.End).IsZero() {
				purchase.MaxEndDate = statusDocument.PotentialRights.End
			}
		}

		return purchase, nil
	}

	return Purchase{}, ErrNotFound
}

// GenerateLicense
func (pManager purchaseManager) GenerateLicense(purchase Purchase) (license.License, error) {
	// Create LCP license
	partialLicense := license.License{}

	log.Println("provuri:" + config.Config.FrontendServer.ProviderUri)

	// Provider
	partialLicense.Provider = config.Config.FrontendServer.ProviderUri

	// User
	encryptedAttrs := []string{"email", "name"}
	partialLicense.User.Email = purchase.User.Email
	partialLicense.User.Name = purchase.User.Name
	partialLicense.User.Id = purchase.User.UUID
	partialLicense.User.Encrypted = encryptedAttrs

	// Encryption
	userKeyValue, err := hex.DecodeString(purchase.User.Password)

	if err != nil {
		return license.License{}, err
	}

	userKey := license.UserKey{}
	userKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	userKey.Hint = purchase.User.Hint
	userKey.Value = userKeyValue
	partialLicense.Encryption.UserKey = userKey

	// Rights
	// FIXME: Do not use harcoded values
	var copy int32
	var print int32
	copy = config.Config.FrontendServer.RightCopy
	print = config.Config.FrontendServer.RightPrint
	userRights := license.UserRights{}
	userRights.Copy = &copy
	userRights.Print = &print

	// Do not include start and end date for a BUY purchase
	if purchase.Type == LOAN {
		userRights.Start = purchase.StartDate
		userRights.End = purchase.EndDate
	}

	partialLicense.Rights = &userRights

	// Encode in json
	jsonBody, err := json.Marshal(partialLicense)

	if err != nil {
		return license.License{}, err
	}

	// Post partial license to LCP
	lcpServerConfig := pManager.config.LcpServer
	var lcpURL string

	if purchase.LicenseUUID == nil {
		lcpURL = lcpServerConfig.PublicBaseUrl + "/contents/" +
			purchase.Publication.UUID + "/licenses"
	} else {
		lcpURL = lcpServerConfig.PublicBaseUrl + "/licenses/" +
			*purchase.LicenseUUID
	}

	log.Println("POST " + lcpURL)

	req, err := http.NewRequest("POST", lcpURL, bytes.NewReader(jsonBody))

	lcpUpdateAuth := pManager.config.LcpUpdateAuth
	if pManager.config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}

	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := lcpClient.Do(req)
	if err != nil {
		return license.License{}, err
	}

	defer resp.Body.Close()

	if (purchase.LicenseUUID == nil && resp.StatusCode != 201) ||
		(purchase.LicenseUUID != nil && resp.StatusCode != 200) {
		// Bad status code
		return license.License{}, errors.New("Bad status code")
	}

	// Decode full license
	fullLicense := license.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&fullLicense)

	if err != nil {
		return license.License{}, errors.New("Unable to decode license")
	}

	// Store license uuid
	purchase.LicenseUUID = &fullLicense.Id
	pManager.Update(purchase)

	if err != nil {
		return license.License{}, errors.New("Unable to update license uuid")
	}

	return fullLicense, nil
}

// GetPartialLicense
func (pManager purchaseManager) GetPartialLicense(purchase Purchase) (license.License, error) {

	if purchase.LicenseUUID == nil {
		return license.License{}, errors.New("No license has been yet delivered")
	}

	// Post partial license to LCP
	lcpServerConfig := pManager.config.LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID
	log.Println("GET " + lcpURL)

	req, err := http.NewRequest("GET", lcpURL, nil)

	lcpUpdateAuth := pManager.config.LcpUpdateAuth
	if pManager.config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}

	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := lcpClient.Do(req)
	if err != nil {
		return license.License{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 206 {
		// Bad status code
		return license.License{}, errors.New("Bad status code")
	}

	// Decode full license
	partialLicense := license.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&partialLicense)

	if err != nil {
		return license.License{}, errors.New("Unable to decode license")
	}

	return partialLicense, nil
}

func (pManager purchaseManager) GetLicenseStatusDocument(purchase Purchase) (licensestatuses.LicenseStatus, error) {
	if purchase.LicenseUUID == nil {
		return licensestatuses.LicenseStatus{}, errors.New("No license has been yet delivered")
	}

	lsdServerConfig := pManager.config.LsdServer
	lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	log.Println("GET " + lsdURL)
	req, err := http.NewRequest("GET", lsdURL, nil)

	lsdAuth := pManager.config.LsdNotifyAuth
	if lsdAuth.Username != "" {
		req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
	}

	req.Header.Add("Content-Type", api.ContentType_JSON)

	var lsdClient = &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := lsdClient.Do(req)
	if err != nil {
		return licensestatuses.LicenseStatus{}, err
	}

	if resp.StatusCode != 200 {
		// Bad status code
		return licensestatuses.LicenseStatus{}, errors.New("Unable to find license")
	}

	// Decode status document
	statusDocument := licensestatuses.LicenseStatus{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&statusDocument)

	if err != nil {
		return licensestatuses.LicenseStatus{}, err
	}

	defer resp.Body.Close()

	return statusDocument, nil
}

func (pManager purchaseManager) GetByLicenseID(licenseID string) (Purchase, error) {
	dbGetByLicenseIDQuery := purchaseManagerQuery + ` WHERE p.license_uuid = ? LIMIT 1`
	dbGetByLicenseID, err := pManager.db.Prepare(dbGetByLicenseIDQuery)
	if err != nil {
		return Purchase{}, err
	}
	defer dbGetByLicenseID.Close()

	records, err := dbGetByLicenseID.Query(licenseID)
	defer records.Close()
	if records.Next() {
		return convertRecordToPurchase(records)
	}

	return Purchase{}, ErrNotFound
}

func (pManager purchaseManager) List(page int, pageNum int) func() (Purchase, error) {
	dbListByUserQuery := purchaseManagerQuery + ` ORDER BY p.transaction_date desc LIMIT ? OFFSET ?`
	dbListByUser, err := pManager.db.Prepare(dbListByUserQuery)

	if err != nil {
		return func() (Purchase, error) { return Purchase{}, err }
	}
	defer dbListByUser.Close()

	records, err := dbListByUser.Query(page, pageNum*page)
	return convertRecordsToPurchases(records)
}

func (pManager purchaseManager) ListByUser(userID int64, page int, pageNum int) func() (Purchase, error) {
	dbListByUserQuery := purchaseManagerQuery + ` WHERE u.id = ?
ORDER BY p.transaction_date desc LIMIT ? OFFSET ?`
	dbListByUser, err := pManager.db.Prepare(dbListByUserQuery)
	if err != nil {
		return func() (Purchase, error) { return Purchase{}, err }
	}
	defer dbListByUser.Close()

	records, err := dbListByUser.Query(userID, page, pageNum*page)
	return convertRecordsToPurchases(records)
}

func (pManager purchaseManager) Add(p Purchase) error {
	add, err := pManager.db.Prepare(`INSERT INTO purchase
	(uuid, publication_id, user_id,
	type, transaction_date,
	start_date, end_date, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, 'ok')`)
	if err != nil {
		return err
	}
	defer add.Close()

	// Fill default values
	if p.TransactionDate.IsZero() {
		p.TransactionDate = time.Now()
	}

	if p.Type == LOAN && p.StartDate == nil {
		p.StartDate = &p.TransactionDate
	}

	// Create uuid
	p.UUID = uuid.NewV4().String()

	_, err = add.Exec(
		p.UUID,
		p.Publication.ID, p.User.ID,
		string(p.Type), p.TransactionDate,
		p.StartDate, p.EndDate)

	return err
}

func (pManager purchaseManager) Update(p Purchase) error {
	// Get original purchase
	origPurchase, err := pManager.Get(p.ID)

	if err != nil {
		return ErrNotFound
	}

	if origPurchase.Status != StatusOk {
		return errors.New("Cannot update an invalid purchase")
	}

	if p.Status == StatusToBeRenewed ||
		p.Status == StatusToBeReturned {

		if p.LicenseUUID == nil {
			return errors.New("Cannot return or renew a purchase when no license has been delivered")
		}

		lsdServerConfig := pManager.config.LsdServer
		lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + *p.LicenseUUID

		if p.Status == StatusToBeRenewed {
			lsdURL += "/renew"

			if p.EndDate != nil {
				lsdURL += "?end=" + p.EndDate.Format(time.RFC3339)
			}

			// Next status if LSD raises no error
			p.Status = StatusOk
		} else if p.Status == StatusToBeReturned {
			lsdURL += "/return"

			// Next status if LSD raises no error
			p.Status = StatusOk
		}

		log.Println("PUT " + lsdURL)
		req, err := http.NewRequest("PUT", lsdURL, nil)

		lsdAuth := pManager.config.LsdNotifyAuth
		if lsdAuth.Username != "" {
			req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
		}

		req.Header.Add("Content-Type", api.ContentType_JSON)

		var lsdClient = &http.Client{
			Timeout: time.Second * 5,
		}

		resp, err := lsdClient.Do(req)
		if err != nil {
			return err
		}

		// FIXME: Check status code

		defer resp.Body.Close()

		// Get new end date from LCP server
		license, err := pManager.GetPartialLicense(origPurchase)

		if err != nil {
			return err
		}

		p.EndDate = license.Rights.End
	} else {
		p.Status = StatusOk
	}

	update, err := pManager.db.Prepare(`UPDATE purchase
	SET license_uuid=?, start_date=?, end_date=?, status=? WHERE id=?`)
	if err != nil {
		return err
	}
	defer update.Close()
	result, err := update.Exec(p.LicenseUUID, p.StartDate, p.EndDate, p.Status, p.ID)
	if changed, err := result.RowsAffected(); err == nil {
		if changed != 1 {
			return ErrNoChange
		}
	}
	return err
}

// Init purchaseManager
func Init(config config.Configuration, db *sql.DB) (i WebPurchase, err error) {
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS purchase (
	id integer NOT NULL,
	uuid varchar(255) NOT NULL,
	publication_id integer NOT NULL,
	user_id integer NOT NULL,
	license_uuid varchar(255) NULL,
	type varchar(32) NOT NULL,
    transaction_date datetime,
	start_date datetime,
	end_date datetime,
	status varchar(255) NOT NULL,
	constraint pk_purchase  primary key(id),
	constraint fk_purchase_publication foreign key (publication_id) references publication(id)
    constraint fk_purchase_user foreign key (user_id) references user(id)
	)`)
	if err != nil {
		log.Println("Error creating purchase table")
		return
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_purchase ON purchase (license_uuid)`)
	if err != nil {
		log.Println("Error creating idx_purchase table")
		return
	}

	i = purchaseManager{config, db}
	return
}
