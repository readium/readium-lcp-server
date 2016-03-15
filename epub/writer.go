package epub

import (
	"archive/zip"
	"io"

	"github.com/readium/readium-lcp-server/xmlenc"
)

const (
	mimetype = "application/epub+zip"
)

type Writer struct {
	w *zip.Writer
}

func (w *Writer) WriteHeader() error {
	return writeMimetype(w.w)
}

func (w *Writer) AddResource(path string, storeMethod uint16) (io.Writer, error) {
	return w.w.CreateHeader(&zip.FileHeader{
		Name:   path,
		Method: storeMethod,
	})
}

func (w *Writer) Copy(r *Resource) error {
	fw, err := w.AddResource(r.Path, r.StorageMethod)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, r.Contents)
	return err
}

func (w *Writer) WriteEncryption(enc *xmlenc.Manifest) error {
	fw, err := w.AddResource("META-INF/encryption.xml", zip.Deflate)
	if err != nil {
		return err
	}

	return enc.Write(fw)

}

func (w *Writer) Close() error {
	return w.w.Close()
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: zip.NewWriter(w),
	}
}

func (ep Epub) Write(dst io.Writer) error {
	w := NewWriter(dst)

	err := w.WriteHeader()
	if err != nil {
		return err
	}

	for _, res := range ep.Resource {
		if res.Path != "mimetype" {
			fw, err := w.AddResource(res.Path, res.StorageMethod)
			if err != nil {
				return err
			}
			_, err = io.Copy(fw, res.Contents)
			if err != nil {
				return err
			}
		}
	}

	if ep.Encryption != nil {
		writeEncryption(ep, w)
	}

	return w.Close()
}

func writeEncryption(ep Epub, w *Writer) error {
	return w.WriteEncryption(ep.Encryption)
}

func writeMimetype(w *zip.Writer) error {
	fh := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	wf.Write([]byte(mimetype))

	return nil
}
