package license

import (
	"database/sql"
	"errors"
)

var NotFound = errors.New("License not found")

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
	_, err := s.db.Exec("INSERT INTO license VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		l.Id, l.User.Id, l.Provider, l.Issued, nil, l.Rights.Print, l.Rights.Copy, l.Rights.Start,
		l.Rights.End, l.Encryption.UserKey.Hint, l.Encryption.UserKey.Check,
		l.Encryption.UserKey.Key.Algorithm, l.ContentId)

	return err
}

func (s *sqlStore) Get(id string) (License, error) {

	var l License
	createForeigns(&l)

	row := s.db.QueryRow(`SELECT id, user_id, provider, issued, updated, rights_print, rights_copy, 
	rights_start, rights_end, user_key_hint, user_key_hash, user_key_algorithm, content_fk FROM license 
	where id = ?`, id)

	err := row.Scan(&l.Id, &l.User.Id, &l.Provider, &l.Issued, &l.Updated,
		&l.Rights.Print, &l.Rights.Copy, &l.Rights.Start, &l.Rights.End,
		&l.Encryption.UserKey.Hint, &l.Encryption.UserKey.Check, &l.Encryption.UserKey.Key.Algorithm,
		&l.ContentId)

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

const tableDef = `CREATE TABLE IF NOT EXISTS license (
	id varchar(255) PRIMARY KEY,
	user_id varchar(255) NOT NULL,
	provider varchar(255) NOT NULL,
	issued datetime NOT NULL,
	updated datetime DEFAULT NULL,
	rights_print int(11) DEFAULT NULL,
	rights_copy int(11) DEFAULT NULL,
	rights_start datetime DEFAULT NULL,
	rights_end datetime DEFAULT NULL,
	user_key_hint text NOT NULL,
  	user_key_hash varchar(64) NOT NULL,
  	user_key_algorithm varchar(255) NOT NULL,
	content_fk varchar(255) NOT NULL)`
