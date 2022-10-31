// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package index

import (
	"database/sql"
	"testing"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/mattn/go-sqlite3"

	"github.com/readium/readium-lcp-server/config"
)

func TestCRUD(t *testing.T) {

	config.Config.LcpServer.Database = "sqlite3://:memory:"
	driver, cnxn := config.GetDatabase(config.Config.LcpServer.Database)
	db, err := sql.Open(driver, cnxn)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		t.Fatal(err)
	}

	idx, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}

	c := Content{ID: "test20",
		EncryptionKey: []byte("1234"),
		Location:      "test.epub",
		Length:        1000,
		Sha256:        "xxxx",
		Type:          "epub"}

	err = idx.Add(c)
	if err != nil {
		t.Fatal(err)
	}
	cbis, err := idx.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != cbis.ID {
		t.Fatal("Failed to Get back the record")
	}

	c.Location = "test.epub"
	err = idx.Update(c)
	if err != nil {
		t.Fatal(err)
	}

	c2 := Content{ID: "test21",
		EncryptionKey: []byte("1234"),
		Location:      "test2.epub",
		Length:        2000,
		Sha256:        "xxxx",
		Type:          "epub"}

	err = idx.Add(c2)
	if err != nil {
		t.Fatal(err)
	}

	fn := idx.List()
	contents := make([]Content, 0)

	for it, err := fn(); err == nil; it, err = fn() {
		contents = append(contents, it)
	}
	if len(contents) != 2 {
		t.Fatal("Failed to List two rows")
	}

}
