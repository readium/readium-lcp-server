// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package license

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/endigo/readium-lcp-server/config"
	"github.com/endigo/readium-lcp-server/sign"
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

	// FIXME: complete this test
	fct := st.ListAll(0, 0)
	if fct == nil {
		t.Errorf("List error")
	}

}

func TestStoreAdd(t *testing.T) {
	config.Config.LcpServer.Database = "sqlite" // FIXME

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st, err := NewSqlStore(db)
	if err != nil {
		t.Fatal(err)
	}

	l := License{}
	contentID := "1234-1234-1234-1234"
	Initialize(contentID, &l)
	setRights(&l)
	err = st.Add(l)
	if err != nil {
		t.Fatal(err)
	}

	l2, err := st.Get(l.ID)
	if err != nil {
		t.Error(err)
	}

	js1, err := sign.Canon(l)
	js2, err2 := sign.Canon(l2)
	if err != nil || err2 != nil || !bytes.Equal(js1, js2) {
		t.Error("Difference between Add and Get")
	}
}

// a rights object is needed before adding a record to the db
// this is copied from lcpserver/api/license.go
// probably this was done in this package and then refactored out, but the test is now broken because of this.
func setRights(lic *License) {

	if lic.Rights == nil {
		lic.Rights = new(UserRights)
	}
	if lic.Rights.Start != nil {
		start := lic.Rights.Start.UTC().Truncate(time.Second)
		lic.Rights.Start = &start
	}
	if lic.Rights.End != nil {
		end := lic.Rights.End.UTC().Truncate(time.Second)
		lic.Rights.End = &end
	}
}
