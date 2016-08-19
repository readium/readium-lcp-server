package transactions

import (
	"database/sql"
	"errors"
	"time"
)

var NotFound = errors.New("Event not found")

type Transactions interface {
	Get(id int) (Event, error)
	Add(e Event) error
	List() func() (Event, error)
	GetByLicenseStatusId(licenseStatusFk int) func() (Event, error)
}

type Event struct {
	Id              int       `json:"-"`
	DeviceName      string    `json:"name"`
	Timestamp       time.Time `json:"timestamp"`
	Type            int       `json:"type"`
	DeviceId        string    `json:"id"`
	LicenseStatusFk string    `json:"-"`
}

type dbTransactions struct {
	db                   *sql.DB
	get                  *sql.Stmt
	add                  *sql.Stmt
	list                 *sql.Stmt
	getbylicensestatusid *sql.Stmt
}

func (i dbTransactions) Get(id int) (Event, error) {
	records, err := i.get.Query(id)
	defer records.Close()
	if records.Next() {
		var e Event
		err = records.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &e.Type, &e.DeviceId, &e.LicenseStatusFk)
		return e, err
	}

	return Event{}, NotFound
}

func (i dbTransactions) Add(e Event) error {
	add, err := i.db.Prepare("INSERT INTO event VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()
	_, err = add.Exec(nil, e.DeviceName, e.Timestamp, e.Type, e.DeviceId, e.LicenseStatusFk)
	return err
}

func (i dbTransactions) List() func() (Event, error) {
	rows, err := i.list.Query()
	if err != nil {
		return func() (Event, error) { return Event{}, err }
	}
	return func() (Event, error) {
		var e Event
		var err error
		if rows.Next() {
			err = rows.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &e.Type, &e.DeviceId, &e.LicenseStatusFk)
		} else {
			rows.Close()
			err = NotFound
		}
		return e, err
	}
}

func (i dbTransactions) GetByLicenseStatusId(licenseStatusFk int) func() (Event, error) {
	rows, err := i.getbylicensestatusid.Query(licenseStatusFk)
	if err != nil {
		return func() (Event, error) { return Event{}, err }
	}
	return func() (Event, error) {
		var e Event
		var err error
		if rows.Next() {
			err = rows.Scan(&e.Id, &e.DeviceName, &e.Timestamp, &e.Type, &e.DeviceId, &e.LicenseStatusFk)
		} else {
			rows.Close()
			err = NotFound
		}
		return e, err
	}
}

func Open(db *sql.DB) (t Transactions, err error) {
	_, err = db.Exec(tableDef)
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT * FROM event WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}
	list, err := db.Prepare("SELECT * FROM event")

	getbylicensestatusid, err := db.Prepare("SELECT * FROM event where license_status_fk = ?")

	if err != nil {
		return
	}

	t = dbTransactions{db, get, nil, list, getbylicensestatusid}
	return
}

const tableDef = `CREATE TABLE IF NOT EXISTS event (
	id integer PRIMARY KEY, 
	device_name varchar(255) NOT NULL,
	timestamp datetime NOT NULL,
	type int NOT NULL,
	device_id varchar(255) NOT NULL,
	license_status_fk int(11) NOT NULL )`
