// Copyright 2020 Readium Foundation. All rights reserved.
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
	"github.com/readium/readium-lcp-server/dbutils"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webuser"
	"github.com/readium/readium-lcp-server/license"
	licensestatuses "github.com/readium/readium-lcp-server/license_statuses"
	uuid "github.com/satori/go.uuid"
)

// ErrNotFound Error is thrown when a purchase is not found
var ErrNotFound = errors.New("purchase not found")

// ErrNoChange is thrown when an update action does not change any rows (not found)
var ErrNoChange = errors.New("no lines were updated")

// WebPurchase defines possible interactions with the db
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

// Purchase struct defines a user in json and database
// PurchaseType: BUY or LOAN
type Purchase struct {
	ID              int64                      `json:"id,omitempty"`
	UUID            string                     `json:"uuid"`
	Publication     webpublication.Publication `json:"publication"`
	User            webuser.User               `json:"user"`
	LicenseUUID     *string                    `json:"licenseUuid,omitempty"`
	Type            string                     `json:"type"`
	TransactionDate time.Time                  `json:"transactionDate,omitempty"`
	StartDate       *time.Time                 `json:"startDate,omitempty"`
	EndDate         *time.Time                 `json:"endDate,omitempty"`
	Status          string                     `json:"status"`
	MaxEndDate      *time.Time                 `json:"maxEndDate,omitempty"`
}

type PurchaseManager struct {
	db               *sql.DB
	dbGetByID        *sql.Stmt
	dbGetByLicenseID *sql.Stmt
	dbList           *sql.Stmt
	dbListByUser     *sql.Stmt
}

func convertRecordsToPurchases(rows *sql.Rows) func() (Purchase, error) {

	return func() (Purchase, error) {
		var err error
		var purchase Purchase
		if rows == nil {
			return purchase, ErrNotFound
		}
		if rows.Next() {
			purchase, err = convertRecordToPurchase(rows)
			if err != nil {
				return purchase, err
			}
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return purchase, err
	}
}

func convertRecordToPurchase(rows *sql.Rows) (Purchase, error) {
	purchase := Purchase{}
	user := webuser.User{}
	pub := webpublication.Publication{}

	err := rows.Scan(
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
	return purchase, nil
}

// Get a purchase using its id
func (pManager PurchaseManager) Get(id int64) (Purchase, error) {

	// note: does not use QueryRow here, as converRecordToPurchase used in converRecordsToPurchase.
	rows, err := pManager.dbGetByID.Query(id)
	if err != nil {
		return Purchase{}, err
	}
	defer rows.Close()

	if rows.Next() {
		purchase, err := convertRecordToPurchase(rows)
		if err != nil {
			return Purchase{}, err
		}

		if purchase.LicenseUUID != nil {
			// Query LSD Server to retrieve max end date (PotentialRights.End)
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

// GetByLicenseID gets a purchase by the associated license id
func (pManager PurchaseManager) GetByLicenseID(licenseID string) (Purchase, error) {

	// note: does not use QueryRow here, as convertRecordToPurchase used in converRecordsToPurchase.
	rows, err := pManager.dbGetByLicenseID.Query(licenseID)
	if err != nil {
		return Purchase{}, err
	}
	defer rows.Close()

	if rows.Next() {
		return convertRecordToPurchase(rows)
		// FIXME: difference with Get(), we don't retrieve the potential end date from the LSD server
	}
	return Purchase{}, ErrNotFound
}

// GenerateOrGetLicense generates a new license associated with a purchase,
// or gets an existing license,
// depending on the value of the license id in the purchase.
func (pManager PurchaseManager) GenerateOrGetLicense(purchase Purchase) (license.License, error) {
	// create a partial license
	partialLicense := license.License{}

	// set the mandatory provider URI
	if config.Config.FrontendServer.ProviderUri == "" {
		return license.License{}, errors.New("mandatory provider URI missing in the configuration")
	}
	partialLicense.Provider = config.Config.FrontendServer.ProviderUri

	// get user info from the purchase info
	encryptedAttrs := []string{"email", "name"}
	partialLicense.User.Email = purchase.User.Email
	partialLicense.User.Name = purchase.User.Name
	partialLicense.User.ID = purchase.User.UUID
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

	// In case of a creation of license, add the user rights
	var copy, print int32
	if purchase.LicenseUUID == nil {
		// in case of undefined conf values for copy and print rights,
		// these rights will be set to zero
		copy = config.Config.FrontendServer.RightCopy
		print = config.Config.FrontendServer.RightPrint
		userRights := license.UserRights{}
		userRights.Copy = &copy
		userRights.Print = &print

		// if this is a loan, include start and end dates from the purchase info
		if purchase.Type == LOAN {
			userRights.Start = purchase.StartDate
			userRights.End = purchase.EndDate
		}
		partialLicense.Rights = &userRights
	}

	// encode in json
	jsonBody, err := json.Marshal(partialLicense)
	if err != nil {
		return license.License{}, err
	}

	// get the url of the lcp server
	lcpServerConfig := config.Config.LcpServer
	var lcpURL string

	if purchase.LicenseUUID == nil {
		// if the purchase contains no license id, generate a new license
		lcpURL = lcpServerConfig.PublicBaseUrl + "/contents/" + purchase.Publication.UUID + "/license"
	} else {
		// if the purchase contains a license id, fetch an existing license
		// note: this will not update the license rights
		lcpURL = lcpServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID
	}
	// message to the console
	log.Println("POST " + lcpURL)

	// add the partial license to the POST request
	req, err := http.NewRequest("POST", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return license.License{}, err
	}
	lcpUpdateAuth := config.Config.LcpUpdateAuth
	if config.Config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// the body is a partial license in json format
	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 10,
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
		return license.License{}, errors.New("the License Server returned an error")
	}

	// decode the full license
	fullLicense := license.License{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&fullLicense)

	if err != nil {
		return license.License{}, errors.New("unable to decode license")
	}

	// store the license id if it was not already set
	if purchase.LicenseUUID == nil {
		purchase.LicenseUUID = &fullLicense.ID
		err = pManager.Update(purchase)
		if err != nil {
			return license.License{}, errors.New("unable to update the license id")
		}
	}

	return fullLicense, nil
}

// GetPartialLicense gets the license associated with a purchase, from the license server
func (pManager PurchaseManager) GetPartialLicense(purchase Purchase) (license.License, error) {

	if purchase.LicenseUUID == nil {
		return license.License{}, errors.New("no license has been yet delivered")
	}

	lcpServerConfig := config.Config.LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID
	// message to the console
	log.Println("GET " + lcpURL)
	// prepare the request
	req, err := http.NewRequest("GET", lcpURL, nil)
	if err != nil {
		return license.License{}, err
	}
	// set credentials
	lcpUpdateAuth := config.Config.LcpUpdateAuth
	if config.Config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// send the request
	var lcpClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := lcpClient.Do(req)
	if err != nil {
		return license.License{}, err
	}
	defer resp.Body.Close()

	// the call must return 206 (partial content) because there is no input partial license
	if resp.StatusCode != 206 {
		// bad status code
		return license.License{}, errors.New("the License Server returned an error")
	}
	// decode the license
	partialLicense := license.License{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&partialLicense)

	if err != nil {
		return license.License{}, errors.New("unable to decode the license")
	}

	return partialLicense, nil
}

// GetLicenseStatusDocument gets a license status document associated with a purchase
func (pManager PurchaseManager) GetLicenseStatusDocument(purchase Purchase) (licensestatuses.LicenseStatus, error) {
	if purchase.LicenseUUID == nil {
		return licensestatuses.LicenseStatus{}, errors.New("no license has been yet delivered")
	}

	lsdServerConfig := config.Config.LsdServer
	lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + *purchase.LicenseUUID + "/status"
	log.Println("GET " + lsdURL)
	req, err := http.NewRequest("GET", lsdURL, nil)
	if err != nil {
		return licensestatuses.LicenseStatus{}, err
	}
	req.Header.Add("Content-Type", api.ContentType_JSON)

	var lsdClient = &http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := lsdClient.Do(req)
	if err != nil {
		return licensestatuses.LicenseStatus{}, err
	}

	if resp.StatusCode != 200 {
		// Bad status code
		return licensestatuses.LicenseStatus{}, errors.New("the License Status Document server returned an error")
	}

	// Decode status document
	statusDocument := licensestatuses.LicenseStatus{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&statusDocument)

	if err != nil {
		return licensestatuses.LicenseStatus{}, err
	}

	defer resp.Body.Close()

	return statusDocument, nil
}

// List all purchases, with pagination
func (pManager PurchaseManager) List(page int, pageNum int) func() (Purchase, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	if driver == "mssql" {
		rows, err = pManager.dbList.Query(pageNum*page, page)
	} else {
		rows, err = pManager.dbList.Query(page, pageNum*page)
	}
	if err != nil {
		log.Printf("Failed to get the full list of purchases: %s", err.Error())
	}
	return convertRecordsToPurchases(rows)
}

// ListByUser lists the purchases of a given user, with pagination
func (pManager PurchaseManager) ListByUser(userID int64, page int, pageNum int) func() (Purchase, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	if driver == "mssql" {
		rows, err = pManager.dbListByUser.Query(userID, pageNum*page, page)
	} else {
		rows, err = pManager.dbListByUser.Query(userID, page, pageNum*page)
	}
	if err != nil {
		log.Printf("Failed to get the user list of purchases: %s", err.Error())
	}
	return convertRecordsToPurchases(rows)
}

// Add a purchase
func (pManager PurchaseManager) Add(p Purchase) error {

	// Fill default values
	if p.TransactionDate.IsZero() {
		p.TransactionDate = time.Now().UTC().Truncate(time.Second)
	}

	if p.Type == LOAN && p.StartDate == nil {
		p.StartDate = &p.TransactionDate
	}

	// Create uuid
	uid, err_u := uuid.NewV4()
	if err_u != nil {
		return err_u
	}
	p.UUID = uid.String()

	_, err := pManager.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database, `INSERT INTO purchase
	(uuid, publication_id, user_id, type, transaction_date, start_date, end_date, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, 'ok')`),
		p.UUID, p.Publication.ID, p.User.ID, string(p.Type), p.TransactionDate, p.StartDate, p.EndDate)

	return err
}

// Update modifies a purchase on a renew or return request
// parameters: a Purchase structure withID,	LicenseUUID, StartDate,	EndDate, Status
// EndDate may be undefined (nil), in which case the lsd server will choose the renew period
func (pManager PurchaseManager) Update(p Purchase) error {
	// Get the original purchase from the db
	origPurchase, err := pManager.Get(p.ID)

	if err != nil {
		return ErrNotFound
	}
	if origPurchase.Status != StatusOk {
		return errors.New("cannot update an invalid purchase")
	}
	if p.Status == StatusToBeRenewed ||
		p.Status == StatusToBeReturned {

		if p.LicenseUUID == nil {
			return errors.New("cannot return or renew a purchase when no license has been delivered")
		}

		lsdServerConfig := config.Config.LsdServer
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
		lsdAuth := config.Config.LsdNotifyAuth
		if lsdAuth.Username != "" {
			req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
		}
		// call the lsd server
		var lsdClient = &http.Client{
			Timeout: time.Second * 10,
		}
		resp, err := lsdClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// if the renew/return request was refused by the License Status server
		if resp.StatusCode != 200 {
			return ErrNoChange
		}

		// get the new end date from the license server
		// FIXME: is there a lighter solution to get the new end date?
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
	result, err := pManager.db.Exec(dbutils.GetParamQuery(config.Config.FrontendServer.Database, `UPDATE purchase SET license_uuid=?, start_date=?, end_date=?, status=? WHERE id=?`),
		p.LicenseUUID, p.StartDate, p.EndDate, p.Status, p.ID)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err == nil {
		if changed != 1 {
			return ErrNoChange
		}
	}
	return err
}

// Init initializes the PurchaseManager
func Init(db *sql.DB) (i WebPurchase, err error) {

	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)

	// if sqlite, create the content table in the frontend db if it does not exist
	if driver == "sqlite3" {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating purchase table")
			return
		}
	}

	selectQuery := `SELECT p.id, p.uuid, p.type, p.transaction_date, p.license_uuid, p.start_date, p.end_date, p.status,
	u.id, u.uuid, u.name, u.email, u.password, u.hint,
	pu.id, pu.uuid, pu.title, pu.status
	FROM purchase p JOIN "user" u ON (p.user_id=u.id) JOIN publication pu ON (p.publication_id=pu.id)`

	var dbGetByID *sql.Stmt
	dbGetByID, err = db.Prepare(selectQuery + dbutils.GetParamQuery(config.Config.FrontendServer.Database, ` WHERE p.id = ?`))
	if err != nil {
		return
	}

	var dbGetByLicenseID *sql.Stmt
	dbGetByLicenseID, err = db.Prepare(selectQuery + dbutils.GetParamQuery(config.Config.FrontendServer.Database, ` WHERE p.license_uuid = ?`))
	if err != nil {
		return
	}

	var dbList *sql.Stmt
	if driver == "mssql" {
		dbList, err = db.Prepare(selectQuery + ` ORDER BY p.transaction_date desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`)
	} else {
		dbList, err = db.Prepare(selectQuery + dbutils.GetParamQuery(config.Config.FrontendServer.Database, ` ORDER BY p.transaction_date desc LIMIT ? OFFSET ?`))
	}
	if err != nil {
		return
	}

	var dbListByUser *sql.Stmt
	if driver == "mssql" {
		dbListByUser, err = db.Prepare(selectQuery + ` WHERE u.id = ? ORDER BY p.transaction_date desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`)
	} else {
		dbListByUser, err = db.Prepare(selectQuery + dbutils.GetParamQuery(config.Config.FrontendServer.Database, ` WHERE u.id = ? ORDER BY p.transaction_date desc LIMIT ? OFFSET ?`))
	}
	if err != nil {
		return
	}

	i = PurchaseManager{db, dbGetByID, dbGetByLicenseID, dbList, dbListByUser}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS purchase (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"publication_id integer NOT NULL," +
	"user_id integer NOT NULL," +
	"license_uuid varchar(255) NULL," +
	"type varchar(32) NOT NULL," +
	"transaction_date datetime," +
	"start_date datetime," +
	"end_date datetime," +
	"status varchar(255) NOT NULL," +
	"FOREIGN KEY (publication_id) REFERENCES publication(id)," +
	"FOREIGN KEY (user_id) REFERENCES user(id)" +
	");" +
	"CREATE INDEX IF NOT EXISTS idx_purchase ON purchase (license_uuid)"
