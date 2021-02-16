// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package index

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/endigo/readium-lcp-server/config"
)

func TestIndexCreation(t *testing.T) {
	config.Config.LcpServer.Database = "sqlite" // FIXME

	db, err := sql.Open("sqlite3", ":memory:")
	idx, err := Open(db)
	if err != nil {
		t.Error("Can't open index")
		t.Error(err)
		t.FailNow()
	}

	c := Content{ID: "test", EncryptionKey: []byte("1234"), Location: "test.epub"}
	err = idx.Add(c)
	if err != nil {
		t.Error(err)
	}
	_, err = idx.Get("test")
	if err != nil {
		t.Error(err)
	}
}
