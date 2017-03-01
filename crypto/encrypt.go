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

package crypto

import (
	"crypto/aes"
	"io"
)
//"github.com/readium/readium-lcp-server/config"
// FOR: config.Config.AES256_CBC_OR_GCM

type Encrypter interface {
	Encrypt(key ContentKey, r io.Reader, w io.Writer) error
	GenerateKey() (ContentKey, error)
	Signature() string
}

type Decrypter interface {
	Decrypt(key ContentKey, r io.Reader, w io.Writer) error
}

func NewAESEncrypter_PUBLICATION_RESOURCES() Encrypter {
	
	return NewAESCBCEncrypter()

	// DISABLED, see https://github.com/readium/readium-lcp-server/issues/109
	// if config.Config.AES256_CBC_OR_GCM == "GCM" {
	// 	return NewAESGCMEncrypter()
	// } else { // default to CBC
	// 	return NewAESCBCEncrypter()
	// }
}

func NewAESEncrypter_CONTENT_KEY() Encrypter {
	// default to CBC
	return NewAESCBCEncrypter()
}

func NewAESEncrypter_USER_KEY_CHECK() Encrypter {
	// default to CBC
	return NewAESEncrypter_CONTENT_KEY()
}

func NewAESEncrypter_FIELDS() Encrypter {
	// default to CBC
	return NewAESEncrypter_CONTENT_KEY()
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
