package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jpbougie/lcpserve/crypto"
	"github.com/jpbougie/lcpserve/epub"
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
	var lic license.License
	var dec *json.Decoder
	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == "application/x-www-form-urlencoded" {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}
	err := dec.Decode(&lic)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	mode := r.PostFormValue("type")
	key := vars["key"]
	if mode == "embedded" {
		item, err := s.Store().Get(key)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		var b bytes.Buffer
		io.Copy(&b, item.Contents())
		zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
		ep, err := epub.Read(zr)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
		var buf bytes.Buffer
		err = grantLicense(&lic, key, false, s, &buf)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		ep.Add("META-INF/licence.lcpl", &buf)
		w.Header().Add("Content-Type", "application/epub+zip")
		ep.Write(w)

	} else {
		w.Header().Add("Content-Type", "application/vnd.readium.lcp.license.1-0+json")
		err = grantLicense(&lic, key, false, s, w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}
	}
}

func grantLicense(l *license.License, key string, embedded bool, s Server, w io.Writer) error {
	p, err := s.Index().Get(key)
	if err != nil {
		return err
	}

	item, err := s.Store().Get(key)
	if err != nil {
		return err
	}

	license.Prepare(l)

	var encryptionKey []byte
	if len(l.Encryption.UserKey.Value) > 0 {
		encryptionKey = l.Encryption.UserKey.Value
		l.Encryption.UserKey.Value = nil
	} else {
		passphrase := l.Encryption.UserKey.ClearValue
		l.Encryption.UserKey.ClearValue = ""
		hash := sha256.Sum256([]byte(passphrase))
		encryptionKey = hash[:]
	}

	l.Encryption.ContentKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
	l.Encryption.ContentKey.Value = encryptKey(p.EncryptionKey, encryptionKey[:])

	l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
	l.Encryption.UserKey.Hint = "Enter your passphrase"

	if !embedded {
		l.Links["publication"] = license.Link{Href: item.PublicUrl(), Type: "application/epub+zip"}
	}
	l.Links["hint"] = license.Link{Href: "http://example.com/hint"}

	encryptFields(l, encryptionKey[:])
	signLicense(l, s.Certificate())

	enc := json.NewEncoder(w)
	enc.Encode(l)

	return nil
}

func encryptFields(l *license.License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(&l.User, toEncrypt)
		crypto.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

func getField(u *license.UserInfo, field string) reflect.Value {
	v := reflect.ValueOf(u).Elem()
	return v.FieldByName(strings.Title(field))
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

func encryptKey(key []byte, kek []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(key)
	crypto.Encrypt(kek[:], in, &out)
	return out.Bytes()
}
