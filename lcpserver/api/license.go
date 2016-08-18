package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/sign"
)

//{
//"content_key": "12345",
//"date": "2013-11-04T01:08:15+01:00",
//"hint": "Enter your email address",
//"hint_url": "http://www.imaginaryebookretailer.com/lcp"
//}

func GenerateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var lic license.License

	err := decodeJsonLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	key := vars["key"]

	w.Header().Add("Content-Type", "application/vnd.readium.lcp.license.1-0+json")
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)

	err = completeLicense(&lic, key, s)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	err = s.Licenses().Add(lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	lic.Encryption.UserKey.Check = nil

	enc := json.NewEncoder(w)
	enc.Encode(lic)
}

func GenerateProtectedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	var lic license.License

	err := decodeJsonLicense(r, &lic)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	key := vars["key"]

	item, err := s.Store().Get(key)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	content, err := s.Index().Get(key)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}
	var b bytes.Buffer
	contents, err := item.Contents()
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	io.Copy(&b, contents)
	zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer

	err = completeLicense(&lic, key, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	lic.Links["publication"] = license.Link{Href: item.PublicUrl(), Type: "application/epub+zip"}
	lic.ContentId = key

	enc := json.NewEncoder(&buf)
	enc.Encode(lic)

	err = s.Licenses().Add(lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	ep.Add("META-INF/license.lcpl", &buf, uint64(buf.Len()))
	w.Header().Add("Content-Type", "application/epub+zip")
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	ep.Write(w)

}

func decodeJsonLicense(r *http.Request, lic *license.License) error {
	var dec *json.Decoder

	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == "application/x-www-form-urlencoded" {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	err := dec.Decode(&lic)

	return err
}

func completeLicense(l *license.License, key string, s Server) error {
	c, err := s.Index().Get(key)
	if err != nil {
		return err
	}

	license.Prepare(l)
	l.ContentId = key

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
	l.Encryption.ContentKey.Value = encryptKey(c.EncryptionKey, encryptionKey[:])

	l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"

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

//ListLicenses returns a JSON struct with information about emitted licenses
// optional GET parameters are "page" (page number) and "per_page" (items par page)
func ListLicenses(w http.ResponseWriter, r *http.Request, s Server) {
	page, err := strconv.ParseInt(r.FormValue("page"), 10, 32)
	if err != nil {
		page = 0 //default starting page
	}
	per_page, err := strconv.ParseInt(r.FormValue("per_page"), 10, 32)
	if err != nil {
		per_page = 30 // default licenses per page
	}
	if page > 0 {
		page -= 1
	} // interface using pageNum starting at page 1 instead of 0 ?
	if page < 0 {
		page = 0
	}
	licenses := make([]license.License, 0)
	log.Println("ListAll(" + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.Licenses().ListAll(int(per_page), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
	}

}

//ListLicenses returns a JSON struct with information about emitted licenses
// content-id is in url
// optional GET parameters are "page" (page number) and "per_page" (items par page)
func ListLicensesForContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	contentId := vars["key"]
	page, err := strconv.ParseInt(r.FormValue("page"), 10, 32)
	if err != nil {
		page = 0 //default starting page
	}
	per_page, err := strconv.ParseInt(r.FormValue("per_page"), 10, 32)
	if err != nil {
		per_page = 30 // default licenses per page
	}
	if page > 0 {
		page -= 1
	} // interface using pageNum starting at page 1 instead of 0 ?
	if page < 0 {
		page = 0
	}
	licenses := make([]license.License, 0)
	log.Println("List(" + contentId + "," + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.Licenses().List(contentId, int(per_page), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		licenses = append(licenses, it)
	}
	if len(licenses) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</licenses/?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</licenses/?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
	}

}
