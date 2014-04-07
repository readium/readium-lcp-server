package license

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/readium/readium-lcp-server/sign"
)

var NotFound = errors.New("Package not found")

type Store interface {
	List() func() (License, error)
	Add(l License) error
	Get(id string) (License, error)
}

type sqlStore struct {
	db *sql.DB
}

func (s *sqlStore) List() func() (License, error) {
	return func() (License, error) {
		return License{}, NotFound
	}
}

func (s *sqlStore) Add(l License) error {
	json, err := sign.Canon(l)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO licenses VALUES (?, ?)", l.Id, json)

	return err
}

func (s *sqlStore) Get(id string) (License, error) {
	var l License
	var buf []uint8

	row := s.db.QueryRow("SELECT data FROM licenses where id = ?", id)

	err := row.Scan(&buf)
	if err != nil {
		return l, err
	}

	b := bytes.NewBuffer(buf)

	dec := json.NewDecoder(b)
	err = dec.Decode(&l)
	if err != nil {
		return l, err
	}

	return l, nil
}

func NewSqlStore(db *sql.DB) (Store, error) {
	_, err := db.Exec(tableDef)
	if err != nil {
		return nil, err
	}

	return &sqlStore{db}, nil
}

const tableDef = `CREATE TABLE IF NOT EXISTS licenses (
	id varchar(255) PRIMARY KEY,
	data blob)`
