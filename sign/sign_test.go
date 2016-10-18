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

package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"math/big"
	"testing"
)

func TestSigningRSA(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("cert/sample_rsa.crt", "cert/sample_rsa.pem")
	if err != nil {
		t.Error("Couldn't load sample certificate ", err)
		t.FailNow()
	}

	signer, err := NewSigner(&cert)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	input := map[string]string{"test": "test"}
	sig, err := signer.Sign(input)

	if expected := "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"; sig.Algorithm != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sig.Algorithm)
	}

	canon, _ := Canon(input)
	hashed := sha256.Sum256(canon)

	if privKey, ok := cert.PrivateKey.(*rsa.PrivateKey); ok {
		if err := rsa.VerifyPKCS1v15(&privKey.PublicKey, crypto.SHA256, hashed[:], sig.Value); err != nil {
			t.Error("Expected the signature to be valid, got", err)
		}
	}
}

func TestSigningECDSA(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("cert/sample_ecdsa.crt", "cert/sample_ecdsa.pem")
	if err != nil {
		t.Error("Couldn't load sample certificate ", err)
		t.FailNow()
	}

	signer, err := NewSigner(&cert)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	input := map[string]string{"test": "test"}
	sig, err := signer.Sign(input)

	if err != nil {
		t.Error(err)
	}

	if expected := "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"; sig.Algorithm != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sig.Algorithm)
	}

	r, s := getParamsFromECDSASignature(sig.Value)

	canon, _ := Canon(input)
	hashed := sha256.Sum256(canon)

	if privKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey); ok {
		if publicKey, ok := privKey.Public().(*ecdsa.PublicKey); ok {
			if !ecdsa.Verify(publicKey, hashed[:], r, s) {
				t.Error("Expected the signature to be valid")
			}
		}
	}
}

func getParamsFromECDSASignature(b []byte) (*big.Int, *big.Int) {
	half := len(b) / 2
	r := big.NewInt(0)
	s := big.NewInt(0)

	r.SetBytes(b[0:half])
	s.SetBytes(b[half:])

	return r, s
}
