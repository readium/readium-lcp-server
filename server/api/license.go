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

package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/sign"

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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, hintFound := lic.Links["hint"]; !hintFound {
		http.Error(w, "hint url not set", http.StatusBadRequest)
		return
	}

	mode := r.PostFormValue("type")
	key := vars["key"]
	if mode == "embedded" {
		item, err := s.Store().Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		indexItem, err := s.Index().Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var b bytes.Buffer
		contents, err := item.Contents()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		io.Copy(&b, contents)
		zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ep, err := epub.Read(zr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var buf bytes.Buffer
		err = grantLicense(&lic, key, false, s, &buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = s.Licenses().Add(lic)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ep.Add("META-INF/license.lcpl", &buf, uint64(buf.Len()))
		w.Header().Add("Content-Type", "application/epub+zip")
		w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, indexItem.Filename))
		ep.Write(w)

	} else {
		w.Header().Add("Content-Type", "application/vnd.readium.lcp.license.1-0+json")
		w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
		err = grantLicense(&lic, key, false, s, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	err = encryptFields(l, encryptionKey[:])
	if err != nil {
		return err
	}
	err = buildKeyCheck(l, encryptionKey[:])
	if err != nil {
		return err
	}
	err = signLicense(l, s.Certificate())
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.Encode(l)

	return nil
}

func buildKeyCheck(l *license.License, key []byte) error {
	var out bytes.Buffer
	err := crypto.Encrypt(key, bytes.NewBufferString(l.Id), &out)
	if err != nil {
		return err
	}
	l.Encryption.UserKey.Check = out.Bytes()
	return nil
}

func encryptFields(l *license.License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(&l.User, toEncrypt)
		err := crypto.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
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
