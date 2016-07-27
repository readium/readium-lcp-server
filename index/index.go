package index

import (
	"database/sql"
	"errors"
)

var NotFound = errors.New("Content not found")

type Index interface {
	Get(id string) (Content, error)
	Add(c Content) error
	List() func() (Content, error)
}

type Content struct {
	Id            string `json:"id"`
	EncryptionKey []byte `json:"encryption_key"`
	Location      string `json:"location"`
}

type dbIndex struct {
	db   *sql.DB
	get  *sql.Stmt
	add  *sql.Stmt
	list *sql.Stmt
}

func (i dbIndex) Get(id string) (Content, error) {
	records, err := i.get.Query(id)
	defer records.Close()
	if records.Next() {
		var c Content
		err = records.Scan(&c.Id, &c.EncryptionKey, &c.Location)
		return c, err
	}

	return Content{}, NotFound
}

func (i dbIndex) Add(c Content) error {
	add, err := i.db.Prepare("INSERT INTO content VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(c.Id, c.EncryptionKey, c.Location)
	return err
}

func (i dbIndex) List() func() (Content, error) {
	rows, err := i.list.Query()
	if err != nil {
		return func() (Content, error) { return Content{}, err }
	}
	return func() (Content, error) {
		var c Content
		var err error
		if rows.Next() {
			err = rows.Scan(&c.Id, &c.EncryptionKey, &c.Location)
		} else {
			rows.Close()
			err = NotFound
		}
		return c, err
	}
}

func Open(db *sql.DB) (i Index, err error) {
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS content (id varchar(255) PRIMARY KEY, encryption_key varchar(64) NOT NULL, location text NOT NULL, FOREIGN KEY(id) REFERENCES license(content_fk))")
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT * FROM content WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}
	list, err := db.Prepare("SELECT * FROM content")
	if err != nil {
		return
	}
	i = dbIndex{db, get, nil, list}
	return
}
