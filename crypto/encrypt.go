package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func Encrypt(key []byte, r io.Reader, w io.Writer) error {
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

func Decrypt(key []byte, r io.Reader, w io.Writer) error {
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

var (
	keywrap_iv = []byte{0xa6, 0xa6, 0xa6, 0xa6,
		0xa6, 0xa6, 0xa6, 0xa6}
)

func KeyWrap(kek []byte, key []byte) []byte {
	cipher, _ := aes.NewCipher(kek)
	n := len(key) / 8
	r := make([]byte, len(keywrap_iv)+len(key))
	a := make([]byte, len(keywrap_iv))

	copy(a, keywrap_iv)
	copy(r[8:], key)

	for j := 0; j < 6; j++ {
		for i := 1; i <= n; i++ {
			out := make([]byte, aes.BlockSize)
			input := make([]byte, aes.BlockSize)
			copy(input, a)
			copy(input[8:], r[i*8:(i+1)*8])
			cipher.Encrypt(out, input)
			t := n*j + i
			copy(a, out[0:8])
			a[7] = a[7] ^ byte(t)
			copy(r[i*8:], out[8:])
		}
	}

	copy(r, a)
	return r
}
