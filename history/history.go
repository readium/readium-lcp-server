package history

import (
	"database/sql"
	"errors"
	"time"

	"github.com/readium/readium-lcp-server/status"
)

var NotFound = errors.New("License Status not found")

type History interface {
	Get(id int) (LicenseStatus, error)
	Add(ls LicenseStatus) error
	List(deviceLimit int64, limit int64, offset int64) func() (LicenseStatus, error)
	GetByLicenseId(id string) (*LicenseStatus, error)
	Update(ls LicenseStatus) error
}

type dbHistory struct {
	db             *sql.DB
	get            *sql.Stmt
	add            *sql.Stmt
	list           *sql.Stmt
	getbylicenseid *sql.Stmt
	update         *sql.Stmt
}

func (i dbHistory) Get(id int) (LicenseStatus, error) {
	var statusDB int64

	records, err := i.get.Query(id)
	defer records.Close()

	if records.Next() {
		ls := LicenseStatus{}
		err = records.Scan(&ls.Id, &statusDB, ls.Updated.License, ls.Updated.Status, &ls.DeviceCount, ls.PotentialRights.End, &ls.LicenseRef)

		if err == nil {
			status.GetStatus(statusDB, &ls.Status)
		}

		return ls, err
	}

	return LicenseStatus{}, NotFound
}

func (i dbHistory) Add(ls LicenseStatus) error {
	add, err := i.db.Prepare("INSERT INTO license_status VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer add.Close()

	statusDB, err := status.SetStatus(ls.Status)

	if err == nil {
		_, err = add.Exec(nil, statusDB, ls.Updated.License, ls.Updated.Status, ls.DeviceCount, ls.PotentialRights.End, ls.LicenseRef)
	}

	return err
}

func (i dbHistory) List(deviceLimit int64, limit int64, offset int64) func() (LicenseStatus, error) {
	rows, err := i.list.Query(deviceLimit, limit, offset)
	if err != nil {
		return func() (LicenseStatus, error) { return LicenseStatus{}, err }
	}
	return func() (LicenseStatus, error) {
		var statusDB int64
		ls := LicenseStatus{}
		ls.PotentialRights = new(PotentialRights)
		ls.Updated = new(Updated)

		var err error
		if rows.Next() {
			err = rows.Scan(&ls.Id, &statusDB, &ls.Updated.License, &ls.Updated.Status, &ls.DeviceCount, &ls.PotentialRights.End, &ls.LicenseRef)

			if err == nil {
				status.GetStatus(statusDB, &ls.Status)
			}
		} else {
			rows.Close()
			err = NotFound
		}
		return ls, err
	}
}

func (i dbHistory) GetByLicenseId(licenseFk string) (*LicenseStatus, error) {
	var statusDB int64
	ls := LicenseStatus{}

	var potentialRightsEnd time.Time
	var licenseUpdate *time.Time
	var statusUpdate *time.Time

	row := i.getbylicenseid.QueryRow(licenseFk)
	err := row.Scan(&ls.Id, &statusDB, &licenseUpdate, &statusUpdate, &ls.DeviceCount, &potentialRightsEnd, &ls.LicenseRef)

	if err == nil {
		status.GetStatus(statusDB, &ls.Status)

		ls.PotentialRights = new(PotentialRights)
		ls.Updated = new(Updated)

		if !potentialRightsEnd.IsZero() {
			ls.PotentialRights.End = potentialRightsEnd
		}

		if licenseUpdate != nil || statusUpdate != nil {
			ls.Updated.Status = statusUpdate
			ls.Updated.License = licenseUpdate
		}
	} else {
		if err == sql.ErrNoRows {
			return nil, err
		}
	}

	return &ls, err
}

func (i dbHistory) Update(ls LicenseStatus) error {

	statusInt, err := status.SetStatus(ls.Status)
	if err != nil {
		return err
	}

	var result sql.Result
	result, err = i.db.Exec("UPDATE license_status SET status=?, license_updated=?, status_updated=?, device_count=?,potential_rights_end=?  WHERE id=?",
		statusInt, ls.Updated.License, ls.Updated.Status, ls.DeviceCount, ls.PotentialRights.End, ls.Id)

	if err == nil {
		if r, _ := result.RowsAffected(); r == 0 {
			return NotFound
		}
	}
	return err
}

func Open(db *sql.DB) (h History, err error) {
	_, err = db.Exec(tableDef)
	if err != nil {
		return
	}
	get, err := db.Prepare("SELECT * FROM license_status WHERE id = ? LIMIT 1")
	if err != nil {
		return
	}

	list, err := db.Prepare(`SELECT * FROM license_status WHERE device_count > ? 
		ORDER BY id DESC LIMIT ? OFFSET ?`)

	getbylicenseid, err := db.Prepare("SELECT * FROM license_status where license_ref = ?")

	if err != nil {
		return
	}
	h = dbHistory{db, get, nil, list, getbylicenseid, nil}
	return
}

const tableDef = `CREATE TABLE IF NOT EXISTS license_status (
  id integer PRIMARY KEY,
  status int(11) NOT NULL,
  license_updated datetime DEFAULT NULL,
  status_updated datetime DEFAULT NULL,
  device_count int(11) DEFAULT NULL,
  potential_rights_end datetime DEFAULT NULL,
  license_ref varchar(255) NOT NULL,
  FOREIGN KEY(id) REFERENCES event(license_status_fk)
)`
