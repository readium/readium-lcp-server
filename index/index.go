package index

import (
	"database/sql"
	"errors"
)

var NotFound = errors.New("Package not found")

type Index interface {
	Get(storageKey string) (Package, error)
	Add(p Package) error
	List() func() (Package, error)
}

type Package struct {
	StorageKey    string `json:"key"`
	EncryptionKey []byte `json:"content_key"`
	Filename      string `json:"filename"`
}

type dbIndex struct {
	db   *sql.DB
	get  *sql.Stmt
	add  *sql.Stmt
	list *sql.Stmt
}

func (i dbIndex) Get(storageKey string) (Package, error) {
	records, err := i.get.Query(storageKey)
	if records.Next() {
		var p Package
		err = records.Scan(&p.StorageKey, &p.EncryptionKey, &p.Filename)
		return p, err
	}

	return Package{}, NotFound
}

func (i dbIndex) Add(p Package) error {
	_, err := i.add.Exec(p.StorageKey, p.EncryptionKey, p.Filename)
	return err
}

func (i dbIndex) List() func() (Package, error) {
	rows, err := i.list.Query()
	if err != nil {
		return func() (Package, error) { return Package{}, err }
	}
	return func() (Package, error) {
		var p Package
		var err error
		if rows.Next() {
			err = rows.Scan(&p.StorageKey, &p.EncryptionKey, &p.Filename)
		} else {
			err = NotFound
		}
		return p, err
	}
}

func Open(db *sql.DB) (i Index, err error) {
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS packages (storage_key varchar(255) PRIMARY KEY, encryption_key blob, filename varchar(255))")
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT * FROM packages WHERE storage_key = ? LIMIT 1")
	if err != nil {
		return
	}
	add, err := db.Prepare("INSERT INTO packages VALUES (?, ?, ?)")
	if err != nil {
		return
	}
	list, err := db.Prepare("SELECT * FROM packages")
	if err != nil {
		return
	}
	i = dbIndex{db, get, add, list}
	return
}
