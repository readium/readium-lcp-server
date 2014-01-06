package api

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"

	"github.com/gorilla/mux"
	"github.com/jpbougie/lcpserve/crypto"
	"github.com/jpbougie/lcpserve/license"
	"github.com/jpbougie/lcpserve/sign"

	"io"
	"net/http"
)

//{
//"content_key": "12345",
//"date": "2013-11-04T01:08:15+01:00",
//"hint": "Enter your email address",
//"hint_url": "http://www.imaginaryebookretailer.com/lcp"
//}

func GrantLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	err := grantLicense(vars["key"], vars["passphrase"], s, w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
}

func grantLicense(key, passphrase string, s Server, w io.Writer) error {
	p, err := s.Index().Get(key)
	if err != nil {
		return err
	}

	item, err := s.Store().Get(key)
	if err != nil {
		return err
	}

	l := license.New()
	l.Encryption.ContentKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#kw-aes256"
	l.Encryption.ContentKey.Value = encryptKey(p.EncryptionKey, passphrase)

	l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	l.Encryption.UserKey.Hint = "Enter your passphrase"

	l.Links["publication"] = license.Link{Href: item.PublicUrl(), Type: "application/epub+zip"}
	l.Links["hint"] = license.Link{Href: "http://example.com/hint"}

	signLicense(&l, s.Certificate())

	enc := json.NewEncoder(w)
	enc.Encode(l)

	return nil
}

func signLicense(l *license.License, cert *tls.Certificate) error {
	sig, err := sign.NewSigner(cert)
	if err != nil {
		return err
	}
	res, err := sig.Sign(l)
	if err != nil {
		return err
	}
	l.Signature = &res

	return nil
}

func encryptKey(key []byte, passphrase string) []byte {
	kek := sha256.Sum256([]byte(passphrase))
	var out bytes.Buffer
	in := bytes.NewReader(key)
	crypto.Encrypt(kek[:], in, &out)
	return out.Bytes()
}
