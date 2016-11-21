package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type cbcEncrypter struct{}

const (
	aes256keyLength = 32 // 256 bits
)

func (e cbcEncrypter) Signature() string {
	return "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
}

func (e cbcEncrypter) GenerateKey() (ContentKey, error) {
	slice, err := GenerateKey(aes256keyLength)
	return ContentKey(slice), err
}

func (e cbcEncrypter) Encrypt(key ContentKey, r io.Reader, w io.Writer) error {
	r = PaddedReader(r, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// generate the IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	// write the IV first
	if _, err = w.Write(iv); err != nil {
		return err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	buffer := make([]byte, aes.BlockSize)
	for _, err = io.ReadFull(r, buffer); err == nil; _, err = io.ReadFull(r, buffer) {
		mode.CryptBlocks(buffer, buffer)
		_, wErr := w.Write(buffer)
		if wErr != nil {
			return wErr
		}
	}

	if err == nil || err == io.EOF {
		return nil
	}

	return err
}

func (c cbcEncrypter) Decrypt(key ContentKey, r io.Reader, w io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	io.Copy(&buffer, r)

	buf := buffer.Bytes()
	iv := buf[:aes.BlockSize]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(buf[aes.BlockSize:], buf[aes.BlockSize:])

	padding := buf[len(buf)-1]
	w.Write(buf[aes.BlockSize : len(buf)-int(padding)])

	return nil
}

func NewAESCBCEncrypter() Encrypter {
	return cbcEncrypter(struct{}{})
}
