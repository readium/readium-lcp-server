// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package webpublication

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"fmt"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/lcpencrypt/encrypt"
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/satori/go.uuid"

	"github.com/Machiel/slugify"
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
	UploadEPUB(*http.Request, http.ResponseWriter, Publication)
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
//
func (pubManager PublicationManager) Get(id int64) (Publication, error) {
	dbGetByID, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication WHERE id = ? LIMIT 1")
	if err != nil {
		return Publication{}, err
	}
	defer dbGetByID.Close()

	records, err := dbGetByID.Query(id)
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
//
func (pubManager PublicationManager) GetByUUID(uuid string) (Publication, error) {
	dbGetByUUID, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication WHERE uuid = ? LIMIT 1")
	if err != nil {
		return Publication{}, err
	}
	defer dbGetByUUID.Close()

	records, err := dbGetByUUID.Query(uuid)
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

// CheckByTitle checks if the publication exists or not, by its title
//
func (pubManager PublicationManager) CheckByTitle(title string) (int64, error) {
	dbGetByTitle, err := pubManager.db.Prepare("SELECT CASE WHEN EXISTS (SELECT * FROM [publication] WHERE title = ?) THEN CAST(1 AS BIT) ELSE CAST(0 AS BIT) END")
	if err != nil {
		return -1, err
	}
	defer dbGetByTitle.Close()

	records, err := dbGetByTitle.Query(title)
	if records.Next() {
		var res int64
		err = records.Scan(
			&res)
		records.Close()

		// returns 1 or 0
		return res, err
	}
	// only if the db query fails
	return -1, ErrNotFound
}

// EncryptEPUB encrypts an EPUB File and sends the content to the LCP server
//
func EncryptEPUB(inputPath string, pub Publication, pubManager PublicationManager) error {
	// generate a new uuid; this will be the content id in the lcp server
	uid, err_u := uuid.NewV4()
	if err_u != nil {
		return err_u
	}
	contentUUID := uid.String()

	// create a temp file in the frontend "encrypted repository"
	outputFilename := contentUUID + ".tmp"
	outputPath := path.Join(pubManager.config.FrontendServer.EncryptedRepository, outputFilename)

	// encrypt the master file found at inputPath, write in the temp file, in the "encrypted repository"
	encryptedEpub, err := encrypt.EncryptEpub(inputPath, outputPath)

	if err != nil {
		// unable to encrypt the master file
		if _, err := os.Stat(inputPath); err == nil {
			os.Remove(inputPath)
		}
		return err
	}

	// prepare the import request to the lcp server
	contentDisposition := slugify.Slugify(pub.Title)
	lcpPublication := apilcp.LcpPublication{}
	lcpPublication.ContentId = contentUUID
	lcpPublication.ContentKey = encryptedEpub.EncryptionKey
	// both frontend and lcp server must understand this path (warning if using Docker containers)
	lcpPublication.Output = outputPath
	lcpPublication.ContentDisposition = &contentDisposition
	lcpPublication.Checksum = &encryptedEpub.Checksum
	lcpPublication.Size = &encryptedEpub.Size

	// json encode the payload
	jsonBody, err := json.Marshal(lcpPublication)
	if err != nil {
		return err
	}
	// send the content to the LCP server
	lcpServerConfig := pubManager.config.LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/contents/" + contentUUID
	log.Println("PUT " + lcpURL)
	req, err := http.NewRequest("PUT", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	// authenticate
	lcpUpdateAuth := pubManager.config.LcpUpdateAuth
	if pubManager.config.LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// set the payload type
	req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

	var lcpClient = &http.Client{
		Timeout: time.Second * 5,
	}
	// sends the import request to the lcp server
	resp, err := lcpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		// error on creation
		return err
	}

	// store the new publication in the db
	// the publication uuid is the lcp db content id.
	pub.UUID = contentUUID
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
// Encrypts a master File and sends the content to the LCP server
//
func (pubManager PublicationManager) Add(pub Publication) error {
	// get the path to the master file
	inputPath := path.Join(
		pubManager.config.FrontendServer.MasterRepository, pub.MasterFilename)

	if _, err := os.Stat(inputPath); err != nil {
		// the master file does not exist
		return err
	}
	// encrypt the EPUB File and send the content to the LCP server
	err := EncryptEPUB(inputPath, pub, pubManager)

	return err
}

// UploadEPUB creates a new EPUB file, namd after a file form parameter.
// a temp file is created then deleted.
//
func (pubManager PublicationManager) UploadEPUB(r *http.Request, w http.ResponseWriter, pub Publication) {

	file, header, err := r.FormFile("file")

	tmpfile, err := ioutil.TempFile("", "example")

	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer os.Remove(tmpfile.Name())

	_, err = io.Copy(tmpfile, file)

	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	// encrypt the EPUB File and send the content to the LCP server
	if err := EncryptEPUB(tmpfile.Name(), pub, pubManager); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, "File uploaded successfully : ")
	fmt.Fprintf(w, header.Filename)
}

// Update updates a publication
// Only the title is updated
//
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
//
func (pubManager PublicationManager) Delete(id int64) error {

	var (
		title string
	)

	fmt.Print("Delete:")
	fmt.Println(id)

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

		// delete the epub file from the master repository
		inputPath := path.Join(pubManager.config.FrontendServer.MasterRepository, title+".epub")

		if _, err := os.Stat(inputPath); err == nil {
			err = os.Remove(inputPath)
			if err != nil {
				return err
			}
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
//
func (pubManager PublicationManager) List(page int, pageNum int) func() (Publication, error) {
	dbList, err := pubManager.db.Prepare("SELECT id, uuid, title, status FROM publication ORDER BY title desc LIMIT ? OFFSET ?")
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
//
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
