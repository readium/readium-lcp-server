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
	"log"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/xmlenc"
)

type PackageReader interface {
	Resources() []Resource
	NewWriter(io.Writer) (PackageWriter, error)
}

type EncryptionProfile string

type PackageWriter interface {
	NewFile(path string, contentType string, storageMethod uint16) (io.WriteCloser, error)
	MarkAsEncrypted(path string, originalSize int64, profile EncryptionProfile, algorithm string)
	Close() error
}

type Resource interface {
	Path() string
	Size() int64
	ContentType() string
	CompressBeforeEncryption() bool
	CanBeEncrypted() bool
	Encrypted() bool
	CopyTo(PackageWriter) error
	Open() (io.ReadCloser, error)
}

func Process(profile EncryptionProfile, encrypter crypto.Encrypter, reader PackageReader, writer PackageWriter) (key crypto.ContentKey, err error) {
	key, err = encrypter.GenerateKey()
	if err != nil {
		log.Println("Error generating a key")
		return
	}

	for _, resource := range reader.Resources() {
		log.Printf("Encrypting %s", resource.Path())
		if !resource.Encrypted() && resource.CanBeEncrypted() {
			err = encryptResource(profile, encrypter, key, resource, writer)
			if err != nil {
				log.Println("Error encrypting " + resource.Path() + ": " + err.Error())
				return
			}
		} else {
			err = resource.CopyTo(writer)
			if err != nil {
				return
			}
		}
	}

	err = writer.Close()

	return
}

func Do(encrypter crypto.Encrypter, ep epub.Epub, w io.Writer) (enc *xmlenc.Manifest, key crypto.ContentKey, err error) {
	key, err = encrypter.GenerateKey()
	if err != nil {
		log.Println("Error generating a key")
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
			err = encryptFile(encrypter, key, ep.Encryption, res, toCompress, ew)
			if err != nil {
				log.Println("Error encrypting " + res.Path + ": " + err.Error())
				return
			}
		} else {
			err = ew.Copy(res)
			if err != nil {
				log.Println("Error copying the file")
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

func encryptResource(profile EncryptionProfile, encrypter crypto.Encrypter, key crypto.ContentKey, resource Resource, packageWriter PackageWriter) error {
	storageMethod := uint16(Deflate)
	mustBeCompressedBeforeEncryption := resource.CompressBeforeEncryption()

	if mustBeCompressedBeforeEncryption {
		storageMethod = NoCompression
	}

	file, err := packageWriter.NewFile(resource.Path(), resource.ContentType(), storageMethod)
	if err != nil {
		return err
	}
	resourceReader, err := resource.Open()
	if err != nil {
		return err
	}
	var reader io.Reader = resourceReader

	if resource.CompressBeforeEncryption() {
		var buffer bytes.Buffer
		w, err := flate.NewWriter(&buffer, 9)
		if err != nil {
			return err
		}

		io.Copy(w, resourceReader)
		resourceReader.Close()
		w.Close()
		reader = ioutil.NopCloser(&buffer)
	}

	err = encrypter.Encrypt(key, reader, file)

	resourceReader.Close()
	file.Close()

	packageWriter.MarkAsEncrypted(resource.Path(), resource.Size(), profile, encrypter.Signature())

	return err
}

func encryptFile(encrypter crypto.Encrypter, key []byte, m *xmlenc.Manifest, file *epub.Resource, compress bool, w *epub.Writer) error {
	data := xmlenc.Data{}
	data.Method.Algorithm = xmlenc.URI(encrypter.Signature())
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
	return encrypter.Encrypt(key, input, fw)
}

func findFile(name string, ep epub.Epub) (*epub.Resource, bool) {
	for _, res := range ep.Resource {
		if res.Path == name {
			return res, true
		}
	}

	return nil, false
}
