// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	uuid "github.com/satori/go.uuid"

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
	Incoming chan *Task
	done     chan struct{}
	store    storage.Store
	idx      index.Index
}

func (p Packager) work() {
	for t := range p.Incoming {
		log.Println("Packager working on an incoming EPUB, encryption task")
		r := Result{}
		p.genKey(&r)
		zr := p.readZip(&r, t.Body, t.Size)
		ep := p.readEpub(&r, zr)
		encrypted, key := p.encrypt(&r, ep)
		p.addToStore(&r, encrypted)
		p.addToIndex(&r, key, t.Name, encrypted, epub.ContentType_EPUB)

		t.Done(r)
	}
}

func (p Packager) genKey(r *Result) {
	if r.Error != nil {
		return
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return
	}
	r.Id = uid.String()
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
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
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

func (p Packager) addToStore(r *Result, info *EncryptedFileInfo) {
	if r.Error != nil {
		return
	}

	_, r.Error = p.store.Add(r.Id, info.File)

	info.File.Close()
	os.Remove(info.File.Name())
}

func (p Packager) addToIndex(r *Result, key []byte, name string, info *EncryptedFileInfo, contentType string) {
	if r.Error != nil {
		return
	}
	r.Error = p.idx.Add(index.Content{Id: r.Id, EncryptionKey: key, Location: name, Length: info.Size, Sha256: info.Sha256, Type: contentType})
}

// NewPackager waits for incoming EPUB files, encrypts them and adds them to the store
func NewPackager(store storage.Store, idx index.Index, concurrency int) *Packager {
	packager := Packager{
		Incoming: make(chan *Task),
		done:     make(chan struct{}),
		store:    store,
		idx:      idx,
	}

	for i := 0; i < concurrency; i++ {
		go packager.work()
	}

	return &packager
}
