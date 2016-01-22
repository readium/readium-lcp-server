package pack

import (
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/xmlenc"
)

func Do(ep epub.Epub) (epub.Epub, []byte, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return ep, nil, err
	}

	ep.Encryption = &xmlenc.Manifest{}
	for _, res := range ep.Resource {
		if canEncrypt(*res, ep) {
			encryptFile(k, ep.Encryption, res, mustCompressBeforeEncryption(*res, ep))
		}
	}
	return ep, k, nil
}

func canEncrypt(file epub.Resource, ep epub.Epub) bool {
	return ep.CanEncrypt(file.Path)
}

// We don't want to compress files that might already be compressed, such
// as multimedia files
func mustCompressBeforeEncryption(file epub.Resource, ep epub.Epub) bool {
	ext := filepath.Ext(file.Path)
	if ext == "" {
		return false
	}

	mimetype := mime.TypeByExtension(ext)

	if mimetype == "" {
		return true
	}

	return !strings.HasPrefix(mimetype, "image") && !strings.HasPrefix(mimetype, "video") && !strings.HasPrefix(mimetype, "audio")
}

func encryptFile(key []byte, m *xmlenc.Manifest, file *epub.Resource, compress bool) error {
	data := xmlenc.Data{}
	data.Method.Algorithm = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
	data.KeyInfo.RetrievalMethod.URI = "license.lcpl#/encryption/content_key"
	data.KeyInfo.RetrievalMethod.Type = "http://readium.org/2014/01/lcp#EncryptedContentKey"
	data.CipherData.CipherReference.URI = xmlenc.URI(file.Path)

	if compress {
		data.Properties = &xmlenc.EncryptionProperties{
			Properties: []xmlenc.EncryptionProperty{
				{Compression: xmlenc.Compression{Method: 8, OriginalLength: file.OriginalSize}},
			},
		}
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
		file.Compressed = true
		w.Close()
		file.ContentsSize = uint64(buf.Len())

		input = ioutil.NopCloser(&buf)
	}
	var output bytes.Buffer
	err := crypto.Encrypt(key, input, &output)

	if err != nil {
		return err
	}

	file.Contents = &output

	return nil
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
