// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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
	GenerateOrGetLicense(purchase Purchase) (license.License, error)
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

type PurchaseManager struct {
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

// Get a purchase using its id
//
func (pManager PurchaseManager) Get(id int64) (Purchase, error) {
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
			// FIXME: calling the lsd server at this point is too heavy: the max end date should be in the db.
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

// GenerateOrGetLicense generates a new license associated with a purchase, ir gets an existing license,
// depending on the calue of the license id in the purchase.
//
func (pManager PurchaseManager) GenerateOrGetLicense(purchase Purchase) (license.License, error) {
	// create a partial license
	partialLicense := license.License{}

	// set the mandatory provider URI
	if config.Config.FrontendServer.ProviderUri == "" {
		return license.License{}, errors.New("Mandatory provider URI missing in the configuration")
	}
	partialLicense.Provider = config.Config.FrontendServer.ProviderUri

	// get user info from the purchase info
	encryptedAttrs := []string{"email", "name"}
	partialLicense.User.Email = purchase.User.Email
	partialLicense.User.Name = purchase.User.Name
	partialLicense.User.Id = purchase.User.UUID
	partialLicense.User.Encrypted = encryptedAttrs

	// get the hashed passphrase from the purchase
	userKeyValue, err := hex.DecodeString(purchase.User.Password)

	if err != nil {
		return license.License{}, err
	}

	userKey := license.UserKey{}
	userKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	userKey.Hint = purchase.User.Hint
	userKey.Value = userKeyValue
	partialLicense.Encryption.UserKey = userKey

	// rights
	var copy int32
	var print int32
	// in case of undefined conf values for copy and print rights,
	// these rights will be set to zero
	copy = config.Config.FrontendServer.RightCopy
	print = config.Config.FrontendServer.RightPrint
	userRights := license.UserRights{}
	userRights.Copy = &copy
	userRights.Print = &print

	// if this is a loan, include start and end date
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

	lcpServerConfig := pManager.config.LcpServer
	var lcpURL string

	if purchase.LicenseUUID == nil {
		// if the purchase contains no license id, generate a new license
		lcpURL = lcpServerConfig.PublicBaseUrl + "/contents/" +
			purchase.Publication.UUID + "/license"
	} else {
		// if the purchase contains a license id, fetch an existing license
		// note: this will not update the license rights
		lcpURL = lcpServerConfig.PublicBaseUrl + "/licenses/" +
			*purchase.LicenseUUID
	}
	// message to the console
	log.Println("POST " + lcpURL)

	// add the partial license to the POST request
	req, err := http.NewRequest("POST", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return license.License{}, err
	}
	lcpUpdateAuth := pManager.config.LcpUpdateAuth
	if pManager.config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// the body is a partial license in json format
	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 5,
	}
	// POST the request
	resp, err := lcpClient.Do(req)
	if err != nil {
		return license.License{}, err
	}

	defer resp.Body.Close()

	// if the status code from the request to the lcp server
	// is neither 201 Created or 200 ok, return an internal error
	if (purchase.LicenseUUID == nil && resp.StatusCode != 201) ||
		(purchase.LicenseUUID != nil && resp.StatusCode != 200) {
		return license.License{}, errors.New("The License Server returned an error")
	}

	// decode the full license
	fullLicense := license.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&fullLicense)

	if err != nil {
		return license.License{}, errors.New("Unable to decode license")
	}

	// store the license id if it was not already set
	if purchase.LicenseUUID == nil {
		purchase.LicenseUUID = &fullLicense.Id
		pManager.Update(purchase)
		if err != nil {
			return license.License{}, errors.New("Unable to update the license id")
		}
	}

	return fullLicense, nil
}

// GetPartialLicense gets the license associated with a purchase, from the license server
//
func (pManager PurchaseManager) GetPartialLicense(purchase Purchase) (license.License, error) {

	if purchase.LicenseUUID == nil {
		return license.License{}, errors.New("No license has been yet delivered")
	}

	lcpServerConfig := pManager.config.LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID
	// message to the console
	log.Println("GET " + lcpURL)
	// prepare the request
	req, err := http.NewRequest("GET", lcpURL, nil)
	if err != nil {
		return license.License{}, err
	}
	// set credentials
	lcpUpdateAuth := pManager.config.LcpUpdateAuth
	if pManager.config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}

	// FIXME: no use
	//req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 5,
	}
	// send the request
	resp, err := lcpClient.Do(req)
	if err != nil {
		return license.License{}, err
	}
	// FIXME: why this Close()?
	defer resp.Body.Close()

	// the call must return 206 (partial content) because there is no input partial license
	if resp.StatusCode != 206 {
		// bad status code
		return license.License{}, errors.New("The License Server returned an error")
	}

	// decode the license
	partialLicense := license.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&partialLicense)

	if err != nil {
		return license.License{}, errors.New("Unable to decode the license")
	}

	return partialLicense, nil
}

// GetLicenseStatusDocument gets a license status document associated with a purchase
//
func (pManager PurchaseManager) GetLicenseStatusDocument(purchase Purchase) (licensestatuses.LicenseStatus, error) {
	if purchase.LicenseUUID == nil {
		return licensestatuses.LicenseStatus{}, errors.New("No license has been yet delivered")
	}

	lsdServerConfig := pManager.config.LsdServer
	lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	log.Println("GET " + lsdURL)
	req, err := http.NewRequest("GET", lsdURL, nil)
	if err != nil {
		return licensestatuses.LicenseStatus{}, err
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
		return licensestatuses.LicenseStatus{}, errors.New("The License Status Document server returned an error")
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

// GetByLicenseID gets a purchase by the associated license id
//
func (pManager PurchaseManager) GetByLicenseID(licenseID string) (Purchase, error) {
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
	// no purchase found
	return Purchase{}, ErrNotFound
}

// List purchases, with pagination
//
func (pManager PurchaseManager) List(page int, pageNum int) func() (Purchase, error) {
	dbListByUserQuery := purchaseManagerQuery + ` ORDER BY p.transaction_date desc LIMIT ? OFFSET ?`
	dbListByUser, err := pManager.db.Prepare(dbListByUserQuery)

	if err != nil {
		return func() (Purchase, error) { return Purchase{}, err }
	}
	defer dbListByUser.Close()

	records, err := dbListByUser.Query(page, pageNum*page)
	return convertRecordsToPurchases(records)
}

// ListByUser: list the purchases of a given user, with pagination
//
func (pManager PurchaseManager) ListByUser(userID int64, page int, pageNum int) func() (Purchase, error) {
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

// Add a purchase
//
func (pManager PurchaseManager) Add(p Purchase) error {
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

// Update modifies a purchase on a renew or return request
// parameters: a Purchase structure withID,	LicenseUUID, StartDate,	EndDate, Status
// EndDate may be undefined (nil), in which case the lsd server will choose the renew period
//
func (pManager PurchaseManager) Update(p Purchase) error {
	// Get the original purchase from the db
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

		// message to the console
		log.Println("PUT " + lsdURL)
		// prepare the request for renew or return to the license status server
		req, err := http.NewRequest("PUT", lsdURL, nil)
		if err != nil {
			return err
		}
		// set credentials
		lsdAuth := pManager.config.LsdNotifyAuth
		if lsdAuth.Username != "" {
			req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
		}
		// call the lsd server
		var lsdClient = &http.Client{
			Timeout: time.Second * 5,
		}
		resp, err := lsdClient.Do(req)
		if err != nil {
			return err
		}
		// FIXME: what is the use of the resp.Body.Close?
		defer resp.Body.Close()

		// get the new end date from the license server
		// FIXME: really needed? heavy...
		license, err := pManager.GetPartialLicense(origPurchase)
		if err != nil {
			return err
		}
		p.EndDate = license.Rights.End
	} else {
		// status is not "to be renewed"
		p.Status = StatusOk
	}
	// update the db with the updated license id, start date, end date, status
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

// Init initializes the PurchaseManager
//
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

	i = PurchaseManager{config, db}
	return
}
