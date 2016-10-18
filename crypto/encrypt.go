// Copyright (c) 2016 Readium Founation
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
