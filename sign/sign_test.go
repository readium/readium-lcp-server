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
