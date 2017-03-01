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