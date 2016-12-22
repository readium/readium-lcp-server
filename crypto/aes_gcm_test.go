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
	"encoding/hex"
	"testing"
)

func TestEncryptGCM(t *testing.T) {
	key, _ := hex.DecodeString("11754cd72aec309bf52f7687212e8957")

	encrypter := NewAESGCMEncrypter()

	data := []byte("The quick brown fox jumps over the lazy dog")

	r := bytes.NewReader(data)
	w := new(bytes.Buffer)

	if err := encrypter.Encrypt(ContentKey(key), r, w); err != nil {
		t.Fatal("Encryption failed", err)
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	out := w.Bytes()
	t.Logf("nonce size: %#v", gcm.NonceSize())
	t.Logf("nonce: %#v", out[0:gcm.NonceSize()])
	t.Logf("ciphertext: %#v", out[gcm.NonceSize():])
	clear := make([]byte, 0)
	clear, err := gcm.Open(clear, out[0:gcm.NonceSize()], out[gcm.NonceSize():], nil)

	if err != nil {
		t.Fatal("Decryption failed", err)
	}

	if diff := bytes.Compare(data, clear); diff != 0 {
		t.Logf("Original: %#v", data)
		t.Logf("After cycle: %#v", clear)
		t.Errorf("Expected encryption-decryption to return original")
	}
}