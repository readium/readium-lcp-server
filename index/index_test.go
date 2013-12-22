package index

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

func TestIndexCreation(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	idx, err := Open(db)
	if err != nil {
		t.Error("Can't open index")
		t.Error(err)
		t.FailNow()
	}

	p := Package{"test", []byte("1234"), "test.epub"}
	err = idx.Add(p)
	if err != nil {
		t.Error(err)
	}
	_, err = idx.Get("test")
	if err != nil {
		t.Error(err)
	}
}
