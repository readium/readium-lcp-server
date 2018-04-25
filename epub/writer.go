/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package epub

import (
	"archive/zip"
	"io"

	"github.com/readium/readium-lcp-server/xmlenc"
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
	fw, err := w.AddResource(EncryptionFile, zip.Deflate)
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

	wf.Write([]byte(ContentTypeEpub))

	return nil
}
