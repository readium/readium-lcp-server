package sign

import (
	"crypto/dsa"
	"crypto/rand"
	"crypto/tls"
	"testing"
)

func TestSigning(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("cert/sample.crt", "cert/sample.pem")
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

	if sig.Algorithm != "http://www.w3.org/2000/09/xmldsig#rsa-sha1" {
		t.Error("Expected 'http://www.w3.org/2000/09/xmldsig#rsa-sha1', got ", sig.Algorithm)
	}
}

func genKey() dsa.PrivateKey {
	var k dsa.PrivateKey

	dsa.GenerateKey(&k, rand.Reader)

	return k
}
