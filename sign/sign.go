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


package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"math"
)

type Signer interface {
	Sign(interface{}) (Signature, error)
}

type Signature struct {
	Certificate []byte `json:"certificate"`
	Value       []byte `json:"value"`
	Algorithm   string `json:"algorithm"`
}

// ECDSA
type ecdsaSigner struct {
	key  *ecdsa.PrivateKey
	cert *tls.Certificate
}

// Used to fill the resulting output according to the XMLDSIG spec
func copyWithLeftPad(dest, src []byte) {
	numPaddingBytes := len(dest) - len(src)
	for i := 0; i < numPaddingBytes; i++ {
		dest[i] = 0
	}
	copy(dest[numPaddingBytes:], src)
}

func (signer *ecdsaSigner) Sign(in interface{}) (sig Signature, err error) {
	plain, err := Canon(in)
	if err != nil {
		return
	}

	hashed := sha256.Sum256(plain)
	r, s, err := ecdsa.Sign(rand.Reader, signer.key, hashed[:])
	if err != nil {
		return
	}

	curveSizeInBytes := int(math.Ceil(float64(signer.key.Curve.Params().BitSize) / 8))

	// The resulting signature is the concatenation of the big-endian octet strings
	// of the r and s parameters, each padded to the byte size of the curve order.
	sig.Value = make([]byte, 2*curveSizeInBytes)
	copyWithLeftPad(sig.Value[0:curveSizeInBytes], r.Bytes())
	copyWithLeftPad(sig.Value[curveSizeInBytes:], s.Bytes())

	sig.Algorithm = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	sig.Certificate = signer.cert.Certificate[0]
	return
}

// RSA
type rsaSigner struct {
	key  *rsa.PrivateKey
	cert *tls.Certificate
}

func (signer *rsaSigner) Sign(in interface{}) (sig Signature, err error) {
	plain, err := Canon(in)
	if err != nil {
		return
	}

	hashed := sha256.Sum256(plain)
	sig.Value, err = rsa.SignPKCS1v15(rand.Reader, signer.key, crypto.SHA256, hashed[:])
	if err != nil {
		return
	}

	sig.Algorithm = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	sig.Certificate = signer.cert.Certificate[0]

	return
}

// Creates a new signer given the certificate type. Currently supports
// RSA (PKCS1v15) and ECDSA (SHA256 is used in both cases)
func NewSigner(certificate *tls.Certificate) (Signer, error) {
	switch k := certificate.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		return &ecdsaSigner{k, certificate}, nil
	case *rsa.PrivateKey:
		return &rsaSigner{k, certificate}, nil
	}

	return nil, errors.New("Unsupported certificate type")
}
