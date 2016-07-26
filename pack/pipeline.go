package pack

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/satori/go.uuid"
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
		r := Result{}
		p.genKey(&r)
		zr := p.readZip(&r, t.Body, t.Size)
		epub := p.readEpub(&r, zr)
		encrypted, key := p.encrypt(&r, epub)
		p.addToStore(&r, encrypted)
		p.addToIndex(&r, key, t.Name)

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

func (p Packager) encrypt(r *Result, ep epub.Epub) (*os.File, []byte) {
	if r.Error != nil {
		return nil, nil
	}

	file, err := ioutil.TempFile(os.TempDir(), "out-readium-lcp")

	if err != nil {
		r.Error = err
		return nil, nil
	}

	_, key, err := Do(ep, file)
	r.Error = err

	file.Seek(0, 0)

	return file, key
}

func (p Packager) addToStore(r *Result, f *os.File) {
	if r.Error != nil {
		return
	}

	_, r.Error = p.store.Add(r.Id, f)

	f.Close()
	os.Remove(f.Name())
}

func (p Packager) addToIndex(r *Result, key []byte, name string) {
	if r.Error != nil {
		return
	}

	r.Error = p.idx.Add(index.Content{r.Id, key, name})
}

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
