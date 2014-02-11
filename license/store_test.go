package license

import (
	"bytes"
	"database/sql"

	"github.com/jpbougie/lcpserve/sign"
	_ "github.com/mattn/go-sqlite3"

	"testing"
)

func TestStoreInit(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewSqlStore(db)
	if err != nil {
		t.Fatal(err)
	}

	it := st.List()
	if _, err := it(); err != NotFound {
		t.Errorf("Didn't expect the iterator to have a value")
	}

}

func TestStoreAdd(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewSqlStore(db)
	if err != nil {
		t.Fatal(err)
	}

	l := New()
	Prepare(&l)
	err = st.Add(l)
	if err != nil {
		t.Error(err)
	}

	l2, err := st.Get(l.Id)
	if err != nil {
		t.Error(err)
	}

	js1, err := sign.Canon(l)
	js2, err2 := sign.Canon(l2)
	if err != nil || err2 != nil || !bytes.Equal(js1, js2) {
		t.Error("Difference between Add and Get")
	}
}
