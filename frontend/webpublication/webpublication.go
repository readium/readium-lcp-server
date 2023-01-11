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
	Upload(multipart.File, string, *Publication) error
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
	db              *sql.DB
	dbGetByID       *sql.Stmt
	dbGetByUUID     *sql.Stmt
	dbCheckByTitle  *sql.Stmt
	dbGetMasterFile *sql.Stmt
	dbList          *sql.Stmt
}

// Get gets a publication by its ID
func (pubManager PublicationManager) Get(id int64) (Publication, error) {

	row := pubManager.dbGetByID.QueryRow(id)
	var pub Publication
	err := row.Scan(
		&pub.ID,
		&pub.UUID,
		&pub.Title,
		&pub.Status)
	return pub, err
}

// GetByUUID returns a publication by its uuid
func (pubManager PublicationManager) GetByUUID(uuid string) (Publication, error) {

	row := pubManager.dbGetByUUID.QueryRow(uuid)
	var pub Publication
	err := row.Scan(
		&pub.ID,
		&pub.UUID,
		&pub.Title,
		&pub.Status)
	return pub, err
}

// CheckByTitle checks if the title of a publication exists or not in the db
func (pubManager PublicationManager) CheckByTitle(title string) (int64, error) {

	row := pubManager.dbCheckByTitle.QueryRow(title)
	var res int64
	err := row.Scan(&res)
	if err != nil {
		return -1, ErrNotFound
	}
	// returns 1 or 0
	return res, err
}

// encryptPublication encrypts a publication, notifies the License Server
// and inserts a record in the database.
func encryptPublication(inputPath string, pub *Publication, pubManager PublicationManager) error {

	var notification *apilcp.LcpPublication

	// encrypt the publication
	// FIXME: work on a direct storage of the output file.
	outputRepo := config.Config.FrontendServer.EncryptedRepository
	empty := ""
	notification, err := encrypt.ProcessEncryption(empty, empty, inputPath, empty, outputRepo, empty, empty, empty)
	if err != nil {
		return err
	}

	// send a notification to the License server
	err = encrypt.NotifyLcpServer(
		notification,
		config.Config.LcpServer.PublicBaseUrl,
		config.Config.LcpUpdateAuth.Username,
		config.Config.LcpUpdateAuth.Password)
	if err != nil {
		return err
	}

	// store the new publication in the db
	// the publication uuid is the lcp db content id.
	pub.UUID = notification.ContentID
	pub.Status = StatusOk
	_, err = pubManager.db.Exec("INSERT INTO publication (uuid, title, status) VALUES ( ?, ?, ?)",
		pub.UUID, pub.Title, pub.Status)

	return err
}

// Add adds a new publication
// Encrypts a master File and notifies the License server
func (pubManager PublicationManager) Add(pub Publication) error {

	// get the path to the master file
	inputPath := filepath.Join(
		config.Config.FrontendServer.MasterRepository, pub.MasterFilename)

	if _, err := os.Stat(inputPath); err != nil {
		// the master file does not exist
		return err
	}

	// encrypt the publication and send a notification to the License server
	err := encryptPublication(inputPath, &pub, pubManager)
	if err != nil {
		return err
	}

	// delete the master file
	err = os.Remove(inputPath)
	return err
}

// Upload creates a new publication, named after a POST form parameter.
// Encrypts a master File and notifies the License server
func (pubManager PublicationManager) Upload(file multipart.File, extension string, pub *Publication) error {

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

	_, err := pubManager.db.Exec("UPDATE publication SET title=?, status=? WHERE id = ?",
		pub.Title, pub.Status, pub.ID)
	return err
}

// Delete deletes a publication, selected by its numeric id
func (pubManager PublicationManager) Delete(id int64) error {

	var title string
	row := pubManager.dbGetMasterFile.QueryRow(id)
	err := row.Scan(&title)
	if err != nil {
		return err
	}

	// delete all purchases relative to this publication
	_, err = pubManager.db.Exec(`DELETE FROM purchase WHERE publication_id=?`, id)
	if err != nil {
		return err
	}

	// delete the publication
	_, err = pubManager.db.Exec("DELETE FROM publication WHERE id = ?", id)
	return err
}

// List lists publications within a given range
// Parameters: page = number of items per page; pageNum = page offset (0 for the first page)
func (pubManager PublicationManager) List(page, pageNum int) func() (Publication, error) {

	var rows *sql.Rows
	var err error
	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)
	if driver == "mssql" {
		rows, err = pubManager.dbList.Query(pageNum*page, page)
	} else {
		rows, err = pubManager.dbList.Query(page, pageNum*page)
	}
	if err != nil {
		return func() (Publication, error) { return Publication{}, err }
	}

	return func() (Publication, error) {
		var pub Publication
		var err error
		if rows.Next() {
			err = rows.Scan(&pub.ID, &pub.UUID, &pub.Title, &pub.Status)
		} else {
			rows.Close()
			err = ErrNotFound
		}
		return pub, err
	}
}

// Init initializes the publication manager
// Creates the publication db table.
func Init(db *sql.DB) (i WebPublication, err error) {

	driver, _ := config.GetDatabase(config.Config.FrontendServer.Database)

	// if sqlite, create the content table in the frontend db if it does not exist
	if driver == "sqlite3" {
		_, err = db.Exec(tableDef)
		if err != nil {
			log.Println("Error creating publication table")
			return
		}
	}

	var dbGetByID *sql.Stmt
	dbGetByID, err = db.Prepare("SELECT id, uuid, title, status FROM publication WHERE id = ?")
	if err != nil {
		return
	}

	var dbGetByUUID *sql.Stmt
	dbGetByUUID, err = db.Prepare("SELECT id, uuid, title, status FROM publication WHERE uuid = ?")
	if err != nil {
		return
	}

	var dbCheckByTitle *sql.Stmt
	dbCheckByTitle, err = db.Prepare("SELECT COUNT(1) FROM publication WHERE title = ?")
	if err != nil {
		return
	}

	var dbGetMasterFile *sql.Stmt
	dbGetMasterFile, err = db.Prepare("SELECT title FROM publication WHERE id = ?")
	if err != nil {
		return
	}

	var dbList *sql.Stmt
	if driver == "mssql" {
		dbList, err = db.Prepare("SELECT id, uuid, title, status FROM publication ORDER BY id desc OFFSET ? ROWS FETCH NEXT ? ROWS ONLY")
	} else {
		dbList, err = db.Prepare("SELECT id, uuid, title, status FROM publication ORDER BY id desc LIMIT ? OFFSET ?")

	}
	if err != nil {
		return
	}

	i = PublicationManager{db, dbGetByID, dbGetByUUID, dbCheckByTitle, dbGetMasterFile, dbList}
	return
}

const tableDef = "CREATE TABLE IF NOT EXISTS publication (" +
	"id integer NOT NULL PRIMARY KEY," +
	"uuid varchar(255) NOT NULL," +
	"title varchar(255) NOT NULL," +
	"status varchar(255) NOT NULL" +
	");" +
	"CREATE INDEX IF NOT EXISTS uuid_index ON publication (uuid);"
