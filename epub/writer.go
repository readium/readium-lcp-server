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
		if (res.File != nil && res.File.Name != "mimetype") || res.FileHeader.Name != "mimetype" {
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
		Name:               "META-INF/encryption.xml",
		UncompressedSize64: uint64(buf.Len()),
		Method:             zip.Deflate,
	}

	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	io.Copy(wf, &buf)
	return nil
}

func writeResource(res Resource, w *zip.Writer) error {
	var fh *zip.FileHeader
	var err error
	if fh = res.FileHeader; fh == nil {
		fh, err = zip.FileInfoHeader(res.File.FileInfo())
		fh.Name = res.File.Name
		if err != nil {
			return err
		}
	}

	var in io.Reader
	if size := res.Output.Len(); size > 0 {
		fh.UncompressedSize64 = uint64(size)
		in = res.Output
	} else {
		in, err = res.File.Open()
		if err != nil {
			return err
		}
	}
	wf, err := w.CreateHeader(fh)
	if err != nil {
		return err
	}

	io.Copy(wf, in)
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
