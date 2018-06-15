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

package model

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/readium/readium-lcp-server/lib/logger"
	"runtime"
	"strings"
)

const (
	LSDLicenseStatusTableName = "lsd_license_statuses"
	LSDTransactionTableName   = "lsd_events"

	LCPLicenseTableName = "lcp_licenses"
	LCPContentTableName = "lcp_contents"

	LUTUserTableName        = "lut_users"
	LUTLicenseViewTableName = "lut_license_views"
	LUTPublicationTableName = "lut_publications"
	LUTPurchaseTableName    = "lut_purchases"
)

type (
	// for transactions
	txStore interface {
		Debug() *gorm.DB
		Set(name string, value interface{}) *gorm.DB
		Create(value interface{}) *gorm.DB
		Delete(value interface{}, where ...interface{}) *gorm.DB
		Where(query interface{}, args ...interface{}) *gorm.DB
	}

	// a logger struct that can be passed to gorm
	dblogger struct {
		logger.StdLogger
	}

	// main struct that holds the database
	dbStore struct {
		db      *gorm.DB
		log     logger.StdLogger
		dialect string
	}

	// similar structs (gets converted to interface)
	userStore             dbStore
	contentStore          dbStore
	dashboardStore        dbStore
	licenseStore          dbStore
	licenseStatusStore    dbStore
	publicationStore      dbStore
	purchaseStore         dbStore
	transactionEventStore dbStore

	// Store interface exposed to the server - each party will transform this interface to another interface
	Store interface {
		Close()
		AutomigrateForFrontend() error
		AutomigrateForLCP() error
		AutomigrateForLSD() error
		Dashboard() DashboardRepository
		User() UserRepository
		License() LicenseRepository
		Publication() PublicationRepository
		Purchase() PurchaseRepository
		Content() ContentRepository
		LicenseStatus() LicenseStatusesRepository
		Transaction() TransactionRepository
	}

	// UserRepository interface for user operations
	UserRepository interface {
		Get(id int64) (*User, error)
		GetByEmail(email string) (*User, error)
		Add(c *User) error
		Update(c *User) error
		Delete(UserID int64) error
		Count() (int64, error)
		FilterCount(emailLike string) (int64, error)
		List(page, pageNum int64) (UsersCollection, error)
		Filter(emailLike string, page, pageNum int64) (UsersCollection, error)
		BulkDelete(ids []int64) error
		ListAll() (UsersCollection, error)
	}

	// DashboardRepository interface for publication db interaction
	DashboardRepository interface {
		GetDashboardInfos() (*Dashboard, error)
		GetDashboardBestSellers() ([]BestSeller, error)
	}

	// PublicationRepository interface for publication db interaction
	PublicationRepository interface {
		Get(id int64) (*Publication, error)
		GetByUUID(uuid string) (*Publication, error)
		Count() (int64, error)
		FilterCount(paramLike string) (int64, error)
		Add(publication *Publication) error
		Update(publication *Publication) error
		Delete(id int64) error
		BulkDelete(pubIds []int64) error
		List(page, pageNum int64) (PublicationsCollection, error)
		ListAll() (PublicationsCollection, error)
		Filter(paramLike string, page, pageNum int64) (PublicationsCollection, error)
		CheckByTitle(title string) (int64, error)
	}

	ContentRepository interface {
		Get(id string) (*Content, error)
		Add(c *Content) error
		Update(c *Content) error
		List() (ContentCollection, error)
	}

	TransactionRepository interface {
		Get(id int64) (*TransactionEvent, error)
		Add(newEvent *TransactionEvent) error
		GetByLicenseStatusId(licenseStatusFk int64) (TransactionEventsCollection, error)
		ListRegisteredDevices(licenseStatusFk int64) (TransactionEventsCollection, error)
		CheckDeviceStatus(licenseStatusFk int64, deviceId string) (Status, error)
	}

	LicenseStatusesRepository interface {
		//Get(id int) (LicenseStatus, error)
		Add(ls *LicenseStatus) error
		Count(deviceLimit int64) (int64, error)
		List(deviceLimit int64, limit int64, offset int64) (LicensesStatusCollection, error)
		ListAll() (LicensesStatusCollection, error)
		GetByLicenseId(id string) (*LicenseStatus, error)
		Update(ls *LicenseStatus) error
	}

	//PurchaseRepository defines possible interactions with DB
	PurchaseRepository interface {
		Get(id int64) (*Purchase, error)
		GetByLicenseID(licenseID string) (*Purchase, error)
		Count() (int64, error)
		List(page int64, pageNum int64) (PurchaseCollection, error)
		Filter(paramLike string, page, pageNum int64) (PurchaseCollection, error)
		FilterCount(paramLike string) (int64, error)
		CountByUser(userID int64) (int64, error)
		ListByUser(userID int64, page int64, pageNum int64) (PurchaseCollection, error)
		Add(p *Purchase) error
		Update(p *Purchase) error
	}

	LicenseRepository interface {
		Count() (int64, error)
		CountForContentId(contentId string) (int64, error)
		List(contentId string, page, pageNum int64) (LicensesCollection, error)
		ListAll(page, pageNum int64) (LicensesCollection, error)
		UpdateRights(l *License) error
		Update(l *License) error
		UpdateLsdStatus(id string, status int32) error
		Add(l *License) error
		Get(id string) (*License, error)
		//from license view
		GetFiltered(filter string) (LicensesCollection, error)
		BulkAdd(licenses LicensesStatusCollection) error
		PurgeDataBase() error
	}
)

func (logger *dblogger) Print(values ...interface{}) {
	if len(values) > 1 {
		level := values[0]
		messages := []interface{}{values[1], ":"}
		if level == "sql" {
			messages = append(messages, values[3])
		}
		pc, _, line, _ := runtime.Caller(8)
		f := runtime.FuncForPC(pc)
		if f != nil && len(values) > 3 {
			// .1) for full source and line
			//logger.Printf("%v [caller %s:%d]", messages, f.Name(), line)
			// .2) for small source and line
			p := strings.Split(values[1].(string), "/")

			parts := strings.Split(f.Name(), "/")
			callerName := parts[len(parts)-1]

			logger.Printf("%s %s [caller %s:%d]", p[len(p)-1:], values[3], callerName, line)
			if len(values) >= 4 {
				v4, ok := values[4].([]interface{})
				if ok {
					for _, val := range v4 {
						logger.Printf("Value : %#v", val)
					}
				} else {
					logger.Printf("Values : %#v", values[4])
				}
			}
		} else {
			logger.Printf("%v", messages)
		}
	}
}

func Transaction(db *gorm.DB, fn func(txStore) error) error {
	tx := db.Set("gorm:save_associations", false).Begin()

	if tx.Error != nil {
		return tx.Error
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *dbStore) Close() {
	s.db.Close()
}

func (s *dbStore) Dashboard() DashboardRepository {
	return (*dashboardStore)(s)
}

func (s *dbStore) User() UserRepository {
	return (*userStore)(s)
}

func (s *dbStore) License() LicenseRepository {
	return (*licenseStore)(s)
}

func (s *dbStore) Content() ContentRepository {
	return (*contentStore)(s)
}

func (s *dbStore) Publication() PublicationRepository {
	return (*publicationStore)(s)
}

func (s *dbStore) Purchase() PurchaseRepository {
	return (*purchaseStore)(s)
}

func (s *dbStore) LicenseStatus() LicenseStatusesRepository {
	return (*licenseStatusStore)(s)
}

func (s *dbStore) Transaction() TransactionRepository {
	return (*transactionEventStore)(s)
}

func (s *dbStore) AutomigrateForLCP() error {
	err := s.db.AutoMigrate(&Content{}, &License{}).Error
	if err != nil {
		return err
	}
	switch s.dialect {
	case "sqlite3":

	case "postgres", "mysql", "mssql":
		s.log.Infof("Creating Index and Foreign Key")
		err = s.db.Model(License{}).AddIndex("contentFKIndex", "content_fk").Error
		if err != nil {
			return err
		}
		err = s.db.Model(License{}).AddForeignKey("content_fk", LCPContentTableName+"(id)", "RESTRICT", "RESTRICT").Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *dbStore) AutomigrateForLSD() error {
	err := s.db.AutoMigrate(&LicenseStatus{}, &TransactionEvent{}).Error
	if err != nil {
		return err
	}
	switch s.dialect {
	case "sqlite3":
		//
	case "postgres", "mysql", "mssql":
		s.log.Infof("Creating Index and Foreign Key")
		err = s.db.Model(TransactionEvent{}).AddIndex("licenseStatusFKIndex", "license_status_fk").Error
		if err != nil {
			return err
		}
		err = s.db.Model(TransactionEvent{}).AddForeignKey("license_status_fk", LSDLicenseStatusTableName+"(id)", "RESTRICT", "RESTRICT").Error
		if err != nil {
			return err
		}
		err = s.db.Model(LicenseStatus{}).AddIndex("license_ref_index", "license_ref").Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *dbStore) AutomigrateForFrontend() error {
	err := s.db.AutoMigrate(&Publication{}, &Purchase{}, &LicenseView{}, &User{}).Error
	if err != nil {
		return err
	}

	switch s.dialect {
	case "sqlite3":

	case "postgres", "mysql", "mssql":
		s.log.Infof("Creating Index and Foreign Key")
		err = s.db.Model(Purchase{}).AddIndex("publicationFKIndex", "publication_id").Error
		if err != nil {
			return err
		}
		err = s.db.Model(Purchase{}).AddForeignKey("publication_id", LUTPublicationTableName+"(id)", "RESTRICT", "RESTRICT").Error
		if err != nil {
			return err
		}
		err = s.db.Model(Purchase{}).AddIndex("userFKIndex", "user_id").Error
		if err != nil {
			return err
		}
		err = s.db.Model(Purchase{}).AddForeignKey("user_id", LUTUserTableName+"(id)", "RESTRICT", "RESTRICT").Error
		if err != nil {
			return err
		}
	}

	return nil
}

func SetupDB(config string, log logger.StdLogger, debugMode bool) (Store, error) {
	var err error

	dialect, cnx := dbFromURI(config)
	if dialect == "error" {
		panic("Incorrect database URI declaration.")
	}

	if dialect == "mysql" && !strings.Contains(cnx, "parseTime") {
		cnx += "?parseTime=true"
	}

	if dialect == "error" {
		return nil, fmt.Errorf("Error : incorrect database configuration : %q", config)
	}

	log.Printf("Database dialect / connex params : %q / %q ", dialect, cnx)
	// database: mysql://root:@tcp(127.0.0.1:3306)/readium_lcp?charset=utf8&parseTime=True&loc=Local
	// database: sqlite3://file:/D:/GoProjects/src/readium-lcp-server/lcp.sqlite?cache=shared&mode=rwc
	// gorm.Open("postgres", "host=myhost port=myport user=gorm dbname=gorm password=mypassword")
	// gorm.Open("mssql", "sqlserver://username:password@localhost:1433?database=dbname")
	db, err := gorm.Open(dialect, cnx)
	if err != nil {
		log.Errorf("Error connecting to database : `%v`", err)
		return nil, err
	}
	db.SetLogger(&dblogger{log})

	if debugMode {
		db.LogMode(true)
	}

	result := &dbStore{db: db, log: log, dialect: dialect}

	err = result.performDialectSpecific()
	if err != nil {
		log.Errorf("Error migrating database %v", err)
		return nil, err
	}

	log.Infof("Gorm ready.")
	return result, nil
}

func (s *dbStore) performDialectSpecific() error {
	switch s.dialect {
	case "sqlite3":
		err := s.db.Exec("PRAGMA journal_mode = WAL").Error
		if err != nil {
			return err
		}
	case "mysql":
		// nothing , so far
	case "postgres":
		// nothing , so far
	case "mssql":
		// nothing , so far
	}
	return nil
}
