package pack

import (
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"path/filepath"
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
	ep.Encryption = &xmlenc.Manifest{}
	for _, res := range ep.Resource {
		if canEncrypt(res, ep) {
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
	ext := filepath.Ext(file.Path)
	if ext == "" {
		return false
	}

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
