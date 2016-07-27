package index

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestIndexCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	idx, err := Open(db)
	if err != nil {
		t.Error("Can't open index")
		t.Error(err)
		t.FailNow()
	}

	c := Content{"test", []byte("1234"), "test.epub"}
	err = idx.Add(c)
	if err != nil {
		t.Error(err)
	}
	_, err = idx.Get("test")
	if err != nil {
		t.Error(err)
	}
}
