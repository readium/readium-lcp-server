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

package storage

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSystemStorage(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "lcpserve_test_store", fmt.Sprintf("%d", rand.New(rand.NewSource(time.Now().UnixNano())).Int()))
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		t.Error("Could not create temp directory for test")
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(dir)

	store := NewFileSystem(dir, "http://localhost/assets")

	item, err := store.Add("test", bytes.NewReader([]byte("test1234")))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if item.Key() != "test" {
		t.Errorf("expected item key to be test, got %s", item.Key())
	}

	// Commented out because store.Info no longer exists.
	//
	// fileInfo, err := store.Info("test")
	// if fileInfo.Name() != "test" {
	// 	t.Errorf("expected item file name to be test, got %s", fileInfo.Name())
	// }
	// if fileInfo.Size() != 8 {
	// 	t.Errorf("expected item file size to be 8, got %d", fileInfo.Size())
	// }

	if item.PublicURL() != "http://localhost/assets/test" {
		t.Errorf("expected item url to be http://localhost/assets/test, got %s", item.PublicURL())
	}

	var buf [8]byte
	contents, err := item.Contents()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadFull(contents, buf[:]); err != nil {
		t.Fatal(err)
	} else {
		if string(buf[:]) != "test1234" {
			t.Error("expected buf to be test1234, got ", string(buf[:]))
		}
	}

	results, err := store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Error("Expected 1 element, got ", len(results))
	}

}
