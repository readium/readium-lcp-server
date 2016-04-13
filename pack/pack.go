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

func Do(encrypter crypto.Encrypter, ep epub.Epub, w io.Writer) (enc *xmlenc.Manifest, key crypto.ContentKey, err error) {
	key, err = encrypter.GenerateKey()
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
			err = encryptFile(encrypter, key, ep.Encryption, res, toCompress, ew)
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
