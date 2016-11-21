package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"io"
	"io/ioutil"
)

type gcmEncrypter struct {
	counter uint64
}

func (e gcmEncrypter) Signature() string {
	return "http://www.w3.org/2009/xmlenc11#aes256-gcm"
}

func (e gcmEncrypter) GenerateKey() (ContentKey, error) {
	slice, err := GenerateKey(aes256keyLength)
	return ContentKey(slice), err
}

func (e *gcmEncrypter) Encrypt(key ContentKey, r io.Reader, w io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	counter := e.counter
	e.counter++

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	binary.BigEndian.PutUint64(nonce, counter)

	data, err := ioutil.ReadAll(r)
	out := gcm.Seal(nonce, nonce, data, nil)

	_, err = w.Write(out)

	return err
}

func NewAESGCMEncrypter() Encrypter {
	return &gcmEncrypter{}
}
