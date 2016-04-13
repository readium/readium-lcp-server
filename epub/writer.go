package epub

import (
	"archive/zip"
	"bytes"
	"io"
)

const (
	mimetype = "application/epub+zip"
)

func (ep Epub) Write(dst io.Writer) error {
	w := zip.NewWriter(dst)

	writeMimetype(w)
	if ep.Encryption != nil {
		writeEncryption(ep, w)
	}

	for _, res := range ep.Resource {
		if res.Path != "mimetype" {
			writeResource(res, w)
		}
	}

	return w.Close()
}

func writeEncryption(ep Epub, w *zip.Writer) error {
	var buf bytes.Buffer
	err := ep.Encryption.Write(&buf)
	if err != nil {
		panic(err)
	}

	fh := &zip.FileHeader{
		Name:   "META-INF/encryption.xml",
		Method: zip.Deflate,
	}

	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	io.Copy(wf, &buf)
	return nil
}

func writeResource(res *Resource, w *zip.Writer) error {
	var err error

	size := res.ContentsSize
	if size == 0 {
		size = res.OriginalSize
	}

	fh := &zip.FileHeader{
		Name:               res.Path,
		UncompressedSize64: size,
	}

	if res.Compressed {
		fh.Method = zip.Store
	} else {
		fh.Method = zip.Deflate
	}

	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	io.Copy(wf, res.Contents)
	return nil
}

func writeMimetype(w *zip.Writer) error {
	fh := &zip.FileHeader{
		Name:               "mimetype",
		UncompressedSize64: uint64(len(mimetype)),
		Method:             zip.Store,
	}
	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	wf.Write([]byte(mimetype))

	return nil
}
