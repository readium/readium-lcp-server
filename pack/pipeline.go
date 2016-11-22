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

package pack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/satori/go.uuid"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/storage"
)

type Source interface {
	Feed(chan<- *Task)
}

type Task struct {
	Name string
	Body io.ReaderAt
	Size int64
	done chan Result
}

type EncryptedFileInfo struct {
	File   *os.File
	Size   int64
	Sha256 string
}

func NewTask(name string, body io.ReaderAt, size int64) *Task {
	return &Task{Name: name, Body: body, Size: size, done: make(chan Result, 1)}
}

type Result struct {
	Error   error
	Id      string
	Elapsed time.Duration
}

func (t *Task) Wait() Result {
	r := <-t.done
	return r
}

func (t *Task) Done(r Result) {
	t.done <- r
}

type ManualSource struct {
	ch chan<- *Task
}

func (s *ManualSource) Feed(ch chan<- *Task) {
	s.ch = ch
}

func (s *ManualSource) Post(t *Task) Result {
	s.ch <- t
	return t.Wait()
}

type Packager struct {
	Incoming  chan *Task
	done      chan struct{}
	store     storage.Store
	idx       index.Index
}

func (p Packager) work() {
	for t := range p.Incoming {
		r := Result{}
		p.genKey(&r)
		zr := p.readZip(&r, t.Body, t.Size)
		epub := p.readEpub(&r, zr)
		encrypted, key := p.encrypt(&r, epub)
		p.addToStore(&r, encrypted.File)
		p.addToIndex(&r, key, t.Name, encrypted.Size, encrypted.Sha256)

		t.Done(r)
	}
}

func (p Packager) genKey(r *Result) {
	if r.Error != nil {
		return
	}

	r.Id = uuid.NewV4().String()
}

func (p Packager) readZip(r *Result, in io.ReaderAt, size int64) *zip.Reader {
	if r.Error != nil {
		return nil
	}

	zr, err := zip.NewReader(in, size)
	r.Error = err
	return zr
}

func (p Packager) readEpub(r *Result, zr *zip.Reader) epub.Epub {
	if r.Error != nil {
		return epub.Epub{}
	}

	ep, err := epub.Read(zr)
	r.Error = err

	return ep
}

func (p Packager) encrypt(r *Result, ep epub.Epub) (*EncryptedFileInfo, []byte) {
	if r.Error != nil {
		return nil, nil
	}
	tmpFile, err := ioutil.TempFile(os.TempDir(), "out-readium-lcp")
	if err != nil {
		r.Error = err
		return nil, nil
	}
	encrypter := crypto.NewAESEncrypter_CONTENT()
	_, key, err := Do(encrypter, ep, tmpFile)
	r.Error = err
	var encryptedFileInfo EncryptedFileInfo
	encryptedFileInfo.File = tmpFile
	//get file length & hash (sha256)
	hasher := sha256.New()
	encryptedFileInfo.File.Seek(0, 0)
	written, err := io.Copy(hasher, encryptedFileInfo.File)
	//hasher.Write(s)
	if err != nil {
		r.Error = err
		return nil, nil
	}
	encryptedFileInfo.Size = written
	encryptedFileInfo.Sha256 = hex.EncodeToString(hasher.Sum(nil))

	encryptedFileInfo.File.Seek(0, 0)
	return &encryptedFileInfo, key
}

func (p Packager) addToStore(r *Result, f *os.File) {
	if r.Error != nil {
		return
	}

	_, r.Error = p.store.Add(r.Id, f)

	f.Close()
	os.Remove(f.Name())
}

func (p Packager) addToIndex(r *Result, key []byte, name string, contentSize int64, contentHash string) {
	if r.Error != nil {
		return
	}

	r.Error = p.idx.Add(index.Content{r.Id, key, name, contentSize, contentHash})
}

func NewPackager(store storage.Store, idx index.Index, concurrency int) *Packager {
	packager := Packager{
		Incoming:  make(chan *Task),
		done:      make(chan struct{}),
		store:     store,
		idx:       idx,
	}

	for i := 0; i < concurrency; i++ {
		go packager.work()
	}

	return &packager
}
