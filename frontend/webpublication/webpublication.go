// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package webpublication

import (
	"database/sql"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/encrypt"
	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
)

// Publication status
const (
	StatusDraft      string = "draft"
	StatusEncrypting string = "encrypting"
	StatusError      string = "error"
	StatusOk         string = "ok"
)

// ErrNotFound error trown when publication is not found
var ErrNotFound = errors.New("Publication not found")

// WebPublication interface for publication db interaction
type WebPublication interface {
	Get(id int64) (Publication, error)
	GetByUUID(uuid string) (Publication, error)
	Add(publication Publication) error
	Update(publication Publication) error
	Delete(id int64) error
	List(page int, pageNum int) func() (Publication, error)
	Upload(multipart.File, string, Publication) error
	CheckByTitle(title string) (int64, error)
}

// Publication struct defines a publication
type Publication struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	Status         string `json:"status"`
	Title          string `json:"title,omitempty"`
	MasterFilename string `json:"masterFilename,omitempty"`
}

// PublicationManager helper
type PublicationManager struct {
	config config.Configuration
	db     *sql.DB
}

// Get gets a publication by its ID
func (pubManager PublicationManager) Get(id int64) (Publication, error) {

	dbGetByID, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication WHERE id = ? LIMIT 1")
	if err != nil {
		return Publication{}, err
	}
	defer dbGetByID.Close()

	records, _ := dbGetByID.Query(id)
	if records.Next() {
		var pub Publication
		err = records.Scan(
			&pub.ID,
			&pub.UUID,
			&pub.Title,
			&pub.Status)
		records.Close()
		return pub, err
	}

	return Publication{}, ErrNotFound
}

// GetByUUID returns a publication by its uuid
func (pubManager PublicationManager) GetByUUID(uuid string) (Publication, error) {

	dbGetByUUID, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication WHERE uuid = ? LIMIT 1")
	if err != nil {
		return Publication{}, err
	}
	defer dbGetByUUID.Close()

	records, _ := dbGetByUUID.Query(uuid)
	if records.Next() {
		var pub Publication
		err = records.Scan(
			&pub.ID,
			&pub.UUID,
			&pub.Title,
			&pub.Status)
		records.Close()
		return pub, err
	}

	return Publication{}, ErrNotFound
}

// CheckByTitle checks if the title of a publication exists or not in the db
func (pubManager PublicationManager) CheckByTitle(title string) (int64, error) {

	dbGetByTitle, err := pubManager.db.Prepare("SELECT COUNT(1) FROM publication WHERE title = ?")
	if err != nil {
		return -1, err
	}
	defer dbGetByTitle.Close()

	records, _ := dbGetByTitle.Query(title)
	if records.Next() {
		var res int64
		err = records.Scan(&res)
		records.Close()

		// returns 1 or 0
		return res, err
	}
	// only if the db query fails
	return -1, ErrNotFound
}

// encryptPublication encrypts a publication, notifies the License Server
// and inserts a record in the database.
func encryptPublication(inputPath string, pub Publication, pubManager PublicationManager) error {

	var notification *apilcp.LcpPublication

	// encrypt the publication
	// FIXME: work on a direct storage of the output file.
	outputRepo := pubManager.config.FrontendServer.EncryptedRepository
	empty := ""
	notification, err := encrypt.ProcessEncryption(empty, inputPath, empty, outputRepo, empty, empty, empty)
	if err != nil {
		return err
	}

	// send a notification to the License server
	err = encrypt.NotifyLcpServer(
		notification,
		pubManager.config.LcpServer.PublicBaseUrl,
		pubManager.config.LcpUpdateAuth.Username,
		pubManager.config.LcpUpdateAuth.Password)
	if err != nil {
		return err
	}

	// store the new publication in the db
	// the publication uuid is the lcp db content id.
	pub.UUID = notification.ContentID
	pub.Status = StatusOk
	dbAdd, err := pubManager.db.Prepare("INSERT INTO publication (uuid, title, status) VALUES ( ?, ?, ?)")
	if err != nil {
		return err
	}
	defer dbAdd.Close()

	_, err = dbAdd.Exec(
		pub.UUID,
		pub.Title,
		pub.Status)

	return err
}

// Add adds a new publication
// Encrypts a master File and notifies the License server
func (pubManager PublicationManager) Add(pub Publication) error {

	// get the path to the master file
	inputPath := filepath.Join(
		pubManager.config.FrontendServer.MasterRepository, pub.MasterFilename)

	if _, err := os.Stat(inputPath); err != nil {
		// the master file does not exist
		return err
	}

	// encrypt the publication and send a notification to the License server
	err := encryptPublication(inputPath, pub, pubManager)
	if err != nil {
		return err
	}

	// delete the master file
	err = os.Remove(inputPath)
	if err != nil {
		return err
	}

	return nil
}

// Upload creates a new publication, named after a POST form parameter.
// Encrypts a master File and notifies the License server
func (pubManager PublicationManager) Upload(file multipart.File, extension string, pub Publication) error {

	// create a temp file in the default directory
	tmpfile, err := ioutil.TempFile("", "uploaded-*"+extension)
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	// copy the request payload to the temp file
	if _, err = io.Copy(tmpfile, file); err != nil {
		return err
	}

	// close the temp file
	if err = tmpfile.Close(); err != nil {
		return err
	}

	// encrypt the publication and send a notification to the License server
	return encryptPublication(tmpfile.Name(), pub, pubManager)
}

// Update updates a publication
// Only the title is updated
func (pubManager PublicationManager) Update(pub Publication) error {

	dbUpdate, err := pubManager.db.Prepare("UPDATE publication SET title=?, status=? WHERE id = ?")
	if err != nil {
		return err
	}
	defer dbUpdate.Close()
	_, err = dbUpdate.Exec(
		pub.Title,
		pub.Status,
		pub.ID)
	if err != nil {
		return err
	}
	return err
}

// Delete deletes a publication, selected by its numeric id
func (pubManager PublicationManager) Delete(id int64) error {

	var title string

	dbGetMasterFile, err := pubManager.db.Prepare("SELECT title FROM publication WHERE id = ?")
	if err != nil {
		return err
	}

	defer dbGetMasterFile.Close()
	result, err := dbGetMasterFile.Query(id)
	if err != nil {
		return err
	}

	if result.Next() {
		err = result.Scan(&title)
		if err != nil {
			return err
		}
	}
	result.Close()

	// delete all purchases relative to this publication
	delPurchases, err := pubManager.db.Prepare(`DELETE FROM purchase WHERE publication_id=?`)
	if err != nil {
		return err
	}
	defer delPurchases.Close()
	if _, err := delPurchases.Exec(id); err != nil {
		return err
	}

	// delete the publication
	dbDelete, err := pubManager.db.Prepare("DELETE FROM publication WHERE id = ?")
	if err != nil {
		return err
	}
	defer dbDelete.Close()
	_, err = dbDelete.Exec(id)
	return err
}

// List lists publications within a given range
// Parameters: page = number of items per page; pageNum = page offset (0 for the first page)
func (pubManager PublicationManager) List(page int, pageNum int) func() (Publication, error) {

	dbList, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication ORDER BY id desc LIMIT ? OFFSET ?")
	if err != nil {
		return func() (Publication, error) { return Publication{}, err }
	}
	defer dbList.Close()
	records, err := dbList.Query(page, pageNum*page)
	if err != nil {
		return func() (Publication, error) { return Publication{}, err }
	}
	return func() (Publication, error) {
		var pub Publication
		if records.Next() {
			err := records.Scan(
				&pub.ID,
				&pub.UUID,
				&pub.Title,
				&pub.Status)
			if err != nil {
				return pub, err
			}

		} else {
			records.Close()
			err = ErrNotFound
		}
		return pub, err
	}
}

// Init initializes the publication manager
// Creates the publication db table.
func Init(config config.Configuration, db *sql.DB) (i WebPublication, err error) {

	// if sqlite, create the content table in the frontend db if it does not exist
	if strings.HasPrefix(config.FrontendServer.Database, "sqlite") {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating publication table")
			return
		}
	}

	i = PublicationManager{config, db}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS publication (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"title varchar(255) NOT NULL," +
	"status varchar(255) NOT NULL" +
	");" +
	"CREATE INDEX IF NOT EXISTS uuid_index ON publication (uuid);"
