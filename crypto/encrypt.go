package crypto

import (
	"crypto/aes"
	"io"
)

type Encrypter interface {
	Encrypt(key ContentKey, r io.Reader, w io.Writer) error
	GenerateKey() (ContentKey, error)
	Signature() string
}

type Decrypter interface {
	Decrypt(key ContentKey, r io.Reader, w io.Writer) error
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
