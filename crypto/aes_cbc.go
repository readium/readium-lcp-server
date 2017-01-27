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
	// W3C padding scheme, not PKCS#7 (see last parameter "insertPadLengthAll" [false] of PaddedReader constructor)
	return "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
}

func (e cbcEncrypter) GenerateKey() (ContentKey, error) {
	slice, err := GenerateKey(aes256keyLength)
	return ContentKey(slice), err
}

func (e cbcEncrypter) Encrypt(key ContentKey, r io.Reader, w io.Writer) error {

	r = PaddedReader(r, aes.BlockSize, false)

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

	padding := buf[len(buf)-1] // padding length valid for both PKCS#7 and W3C schemes
	w.Write(buf[aes.BlockSize : len(buf)-int(padding)])

	return nil
}

func NewAESCBCEncrypter() Encrypter {
	return cbcEncrypter(struct{}{})
}
