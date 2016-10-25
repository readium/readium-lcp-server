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
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/xmlenc"
)

func Do(ep epub.Epub, w io.Writer) (enc *xmlenc.Manifest, key []byte, err error) {
	key, err = crypto.GenerateKey()
	if err != nil {
		return
	}

	ew := epub.NewWriter(w)
	ew.WriteHeader()
	if ep.Encryption == nil {
		ep.Encryption = &xmlenc.Manifest{}
	}

	for _, res := range ep.Resource {
		if _, alreadyEncrypted := ep.Encryption.DataForFile(res.Path); !alreadyEncrypted && canEncrypt(res, ep) {
			toCompress := mustCompressBeforeEncryption(*res, ep)
			err = encryptFile(key, ep.Encryption, res, toCompress, ew)
			if err != nil {
				return
			}
		} else {
			err = ew.Copy(res)
			if err != nil {
				return
			}
		}
	}

	ew.WriteEncryption(ep.Encryption)

	return ep.Encryption, key, ew.Close()
}

// We don't want to compress files that might already be compressed, such
// as multimedia files
func mustCompressBeforeEncryption(file epub.Resource, ep epub.Epub) bool {
	mimetype := file.ContentType

	if mimetype == "" {
		return true
	}

	return !strings.HasPrefix(mimetype, "image") && !strings.HasPrefix(mimetype, "video") && !strings.HasPrefix(mimetype, "audio")
}

const (
	NoCompression = 0
	Deflate       = 8
)

func canEncrypt(file *epub.Resource, ep epub.Epub) bool {
	return ep.CanEncrypt(file.Path)
}

func encryptFile(key []byte, m *xmlenc.Manifest, file *epub.Resource, compress bool, w *epub.Writer) error {
	data := xmlenc.Data{}
	data.Method.Algorithm = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
	data.KeyInfo = &xmlenc.KeyInfo{}
	data.KeyInfo.RetrievalMethod.URI = "license.lcpl#/encryption/content_key"
	data.KeyInfo.RetrievalMethod.Type = "http://readium.org/2014/01/lcp#EncryptedContentKey"
	data.CipherData.CipherReference.URI = xmlenc.URI(file.Path)

	method := NoCompression
	if compress {
		method = Deflate
	}

	file.StorageMethod = NoCompression

	data.Properties = &xmlenc.EncryptionProperties{
		Properties: []xmlenc.EncryptionProperty{
			{Compression: xmlenc.Compression{Method: method, OriginalLength: file.OriginalSize}},
		},
	}

	m.Data = append(m.Data, data)

	input := file.Contents

	if compress {
		var buf bytes.Buffer
		w, err := flate.NewWriter(&buf, 9)
		if err != nil {
			return err
		}

		io.Copy(w, file.Contents)
		w.Close()
		file.ContentsSize = uint64(buf.Len())

		input = ioutil.NopCloser(&buf)
	}

	fw, err := w.AddResource(file.Path, file.StorageMethod)
	if err != nil {
		return err
	}
	return crypto.Encrypt(key, input, fw)
}

func Undo(key []byte, ep epub.Epub) (epub.Epub, error) {
	for _, data := range ep.Encryption.Data {
		if res, ok := findFile(string(data.CipherData.CipherReference.URI), ep); ok {
			var buf bytes.Buffer
			crypto.Decrypt(key, res.Contents, &buf)
			res.Contents = &buf
		}
	}

	ep.Encryption = nil

	return ep, nil
}

func findFile(name string, ep epub.Epub) (*epub.Resource, bool) {
	for _, res := range ep.Resource {
		if res.Path == name {
			return res, true
		}
	}

	return nil, false
}
