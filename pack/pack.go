// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package pack

import (
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/xmlenc"
)

// PackageReader is an interface
type PackageReader interface {
	Resources() []Resource
	NewWriter(io.Writer) (PackageWriter, error)
}

// PackageWriter is an interface
type PackageWriter interface {
	NewFile(path string, contentType string, storageMethod uint16) (io.WriteCloser, error)
	MarkAsEncrypted(path string, originalSize int64, profile license.EncryptionProfile, algorithm string)
	Close() error
}

// Resource is an interface
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

// Process copies resources from the source to the destination package, after encryption if needed.
func Process(profile license.EncryptionProfile, encrypter crypto.Encrypter, reader PackageReader, writer PackageWriter) (key crypto.ContentKey, err error) {

	// generate an encryption key
	key, err = encrypter.GenerateKey()
	if err != nil {
		log.Println("Error generating an encryption key")
		return
	}

	// loop through the resources of the source package, encrypt them if needed, copy them into the dest package
	for _, resource := range reader.Resources() {
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

// Do encrypts when necessary the resources of an EPUB package
// Note: It calls encryptFile
// It is currently called only for EPUB files
// FIXME: try to merge Process() and Do()
func Do(encrypter crypto.Encrypter, ep epub.Epub, w io.Writer) (enc *xmlenc.Manifest, key crypto.ContentKey, err error) {

	// generate an encryption key
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

// mustCompressBeforeEncryption checks is a resource must be compressed before encryption.
// We don't want to compress files if that might cause streaming issues.
// The test is applied on the resource media-type; image, video and audio are stored without compression.
func mustCompressBeforeEncryption(file epub.Resource, ep epub.Epub) bool {

	mimetype := file.ContentType

	if mimetype == "" {
		return true
	}

	return !strings.HasPrefix(mimetype, "image") && !strings.HasPrefix(mimetype, "video") && !strings.HasPrefix(mimetype, "audio")
}

// NoCompression means Store
const (
	NoCompression = 0
	Deflate       = 8
)

// canEncrypt checks if a resource should be encrypted
func canEncrypt(file *epub.Resource, ep epub.Epub) bool {
	return ep.CanEncrypt(file.Path)
}

// encryptResource encrypts a resource in a Readium Package
func encryptResource(profile license.EncryptionProfile, encrypter crypto.Encrypter, key crypto.ContentKey, resource Resource, packageWriter PackageWriter) error {

	storageMethod := uint16(Deflate)

	// FIXME: this is currently always set to false
	mustBeCompressedBeforeEncryption := resource.CompressBeforeEncryption()

	if mustBeCompressedBeforeEncryption {
		// no further compression when zipping if the resource has already been deflated
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

	if mustBeCompressedBeforeEncryption {
		var buffer bytes.Buffer
		deflateWriter, err := flate.NewWriter(&buffer, 9)
		if err != nil {
			return err
		}

		io.Copy(deflateWriter, resourceReader)
		resourceReader.Close()
		deflateWriter.Close()
		reader = ioutil.NopCloser(&buffer)
	}

	err = encrypter.Encrypt(key, reader, file)

	resourceReader.Close()
	file.Close()

	packageWriter.MarkAsEncrypted(resource.Path(), resource.Size(), profile, encrypter.Signature())

	return err
}

// encryptFile encrypts a file in an EPUB package
func encryptFile(encrypter crypto.Encrypter, key []byte, m *xmlenc.Manifest, file *epub.Resource, compress bool, w *epub.Writer) error {

	data := xmlenc.Data{}
	data.Method.Algorithm = xmlenc.URI(encrypter.Signature())
	data.KeyInfo = &xmlenc.KeyInfo{}
	data.KeyInfo.RetrievalMethod.URI = "license.lcpl#/encryption/content_key"
	data.KeyInfo.RetrievalMethod.Type = "http://readium.org/2014/01/lcp#EncryptedContentKey"

	uri, err := url.Parse(file.Path)
	if err != nil {
		return err
	}
	data.CipherData.CipherReference.URI = xmlenc.URI(uri.EscapedPath())

	method := NoCompression
	if compress {
		method = Deflate
	}

	// set the storage method to Deflate or NoCompression
	file.StorageMethod = uint16(method)

	data.Properties = &xmlenc.EncryptionProperties{
		Properties: []xmlenc.EncryptionProperty{
			{Compression: xmlenc.Compression{Method: method, OriginalLength: file.OriginalSize}},
		},
	}

	m.Data = append(m.Data, data)

	input := file.Contents

	if compress {
		var buf bytes.Buffer
		deflateWriter, err := flate.NewWriter(&buf, 9)
		if err != nil {
			return err
		}
		io.Copy(deflateWriter, file.Contents)
		deflateWriter.Close()
		file.ContentsSize = uint64(buf.Len())

		input = ioutil.NopCloser(&buf)
	}

	fw, err := w.AddResource(file.Path, file.StorageMethod)
	if err != nil {
		return err
	}
	return encrypter.Encrypt(key, input, fw)
}

// findFile finds a file in an EPUB object
func findFile(name string, ep epub.Epub) (*epub.Resource, bool) {

	for _, res := range ep.Resource {
		if res.Path == name {
			return res, true
		}
	}

	return nil, false
}
