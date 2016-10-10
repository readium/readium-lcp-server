package apilcp

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
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
func GetLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenceId := vars["key"]
	// search existing license using key
	var ExistingLicense license.License
	ExistingLicense, e := s.Licenses().Get(licenceId)
	if e != nil {
		if e == license.NotFound {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: e.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: e.Error()}, http.StatusBadRequest)
		}
		return
	}
	var lic license.License
	err := DecodeJsonLicense(r, &lic)
	if err != nil { // no or incorrect (json) license found in body
		// just send partial license
		w.Header().Add("Content-Type", license.ContentType)
		w.WriteHeader(http.StatusPartialContent)
		//delete some sensitive data from license
		ExistingLicense.Encryption.UserKey.Check = nil
		ExistingLicense.Encryption.UserKey.Value = nil
		ExistingLicense.Encryption.UserKey.Hint = ""
		ExistingLicense.Encryption.UserKey.ClearValue = ""
		ExistingLicense.Encryption.UserKey.Key.Algorithm = ""
		ExistingLicense.Encryption.Profile = ""

		enc := json.NewEncoder(w)
		enc.Encode(ExistingLicense)

		return
	} else { // add information to license , sign and return (real) License
		if lic.User.Email == "" {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "User information must be passed in INPUT"}, http.StatusBadRequest)
			return
		}
		ExistingLicense.User = lic.User
		content, err := s.Index().Get(ExistingLicense.ContentId)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		ExistingLicense.Encryption.ContentKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
		ExistingLicense.Encryption.ContentKey.Value = encryptKey(content.EncryptionKey, ExistingLicense.Encryption.UserKey.Value) //use old UserKey.Value
		ExistingLicense.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"
		err = buildKeyCheck(&ExistingLicense, ExistingLicense.Encryption.UserKey.Value)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		err = signLicense(&ExistingLicense, s.Certificate())
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", license.ContentType)
		w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)
		enc := json.NewEncoder(w)
		enc.Encode(ExistingLicense)
		return
	}
}

func UpdateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	problem.Error(w, r, problem.Problem{Type: "about:blank"}, http.StatusNotImplemented)
	vars := mux.Vars(r)
	licenceId := vars["key"]
	// search existing license using key
	var ExistingLicense license.License
	ExistingLicense, e := s.Licenses().Get(licenceId)
	if e != nil {
		if e == license.NotFound {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: license.NotFound.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: e.Error()}, http.StatusBadRequest)
		}
		return
	}
	var lic license.License
	err := DecodeJsonLicense(r, &lic)
	if err != nil { // no or incorrect (json) license found in body
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if lic.Id != licenceId {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "Different license IDs"}, http.StatusNotFound)
		return
	}
	// update rights of license in database / verify validity of lic / existingLicense
	if lic.Provider != "" {
		ExistingLicense.Provider = lic.Provider
	}
	if lic.Rights.Copy != nil {
		ExistingLicense.Rights.Copy = lic.Rights.Copy
	}
	if lic.Rights.Print != nil {
		ExistingLicense.Rights.Print = lic.Rights.Print
	}
	if lic.Rights.Start != nil {
		ExistingLicense.Rights.Start = lic.Rights.Start
	}
	if lic.Rights.End != nil {
		ExistingLicense.Rights.End = lic.Rights.End
	}
	if lic.Encryption.UserKey.Hint != "" {
		ExistingLicense.Encryption.UserKey.Hint = lic.Encryption.UserKey.Hint
	}
	if lic.ContentId != "" { //change content
		ExistingLicense.ContentId = lic.ContentId
	}
	err = s.Licenses().Update(ExistingLicense)
	if err != nil { // no or incorrect (json) license found in body
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// go on and GET license io to return the updated license
	GetLicense(w, r, s)
}

func UpdateRightsLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	licenceId := vars["key"]
	// search existing license using key
	var ExistingLicense license.License
	ExistingLicense, e := s.Licenses().Get(licenceId)
	if e != nil {
		if e == license.NotFound {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: license.NotFound.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: e.Error()}, http.StatusBadRequest)
		}
		return
	}
	var lic license.License
	err := DecodeJsonLicense(r, &lic)
	if err != nil { // no or incorrect (json) license found in body
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if lic.Id != licenceId {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "Different license IDs"}, http.StatusNotFound)
		return
	}
	// update rights of license in database
	if lic.Rights.Copy != nil {
		ExistingLicense.Rights.Copy = lic.Rights.Copy
	}
	if lic.Rights.Print != nil {
		ExistingLicense.Rights.Print = lic.Rights.Print
	}
	if lic.Rights.Start != nil {
		ExistingLicense.Rights.Start = lic.Rights.Start
	}
	if lic.Rights.End != nil {
		ExistingLicense.Rights.End = lic.Rights.End
	}
	err = s.Licenses().UpdateRights(ExistingLicense)
	if err != nil { // no or incorrect (json) license found in body
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	// go on to GET license io to return the existing license
	GetLicense(w, r, s)
}

func GenerateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var lic license.License

	err := DecodeJsonLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	key := vars["key"]

	w.Header().Add("Content-Type", license.ContentType)
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

	//lic.Encryption.UserKey.Check = nil

	enc := json.NewEncoder(w)
	enc.Encode(lic)
}

func GenerateProtectedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	key := vars["key"]

	var lic license.License

	err := DecodeJsonLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	if key == "" {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "No content id", Instance: key}, http.StatusBadRequest)

		return
	}

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

	//lic.Links["publication"] = license.Link{Href: item.PublicUrl(), Type: "application/epub+zip"}
	//lic.ContentId = key

	err = completeLicense(&lic, key, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(&buf)
	enc.Encode(lic)

	err = s.Licenses().Add(lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error(), Instance: key}, http.StatusInternalServerError)
		return
	}

	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add("META-INF/license.lcpl", &buf2, uint64(buf2.Len()))
	w.Header().Add("Content-Type", "application/epub+zip")
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	ep.Write(w)

}

func DecodeJsonLicense(r *http.Request, lic *license.License) error {
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
	//verify that mandatory(hint & publication ) links are present in the License
	if _, present := l.Links["hint"]; !present {
		hint := new(license.Link)
		hint.Href, present = config.Config.License.Links["hint"]
		//hint.Type =
		if !present {
			return errors.New("No hint link present in partial license nor config")
		}
	}

	if _, present := l.Links["publication"]; !present {
		publication := new(license.Link)
		publication.Href, present = config.Config.License.Links["publication"]
		if !present {
			return errors.New("No publication link present in partial license nor config")
		}
		//publication.Type = ?? , other information about encrypted file (md5 hash ?)
		l.Links["publication"] = *publication
	}
	// replace {publication_id} in template link
	publicationLink := strings.Replace(l.Links["publication"].Href, "{publication_id}", c.Id, 1)
	publicationLink = strings.Replace(publicationLink, "{publication_loc}", c.Location, 1)
	if publicationLink != l.Links["publication"].Href {
		publication := new(license.Link)
		publication.Href = publicationLink
		l.Links["publication"] = *publication
	}

	// finalize by ensuring type field is correctly set
	publi := new(license.Link)
	publi.Href = l.Links["publication"].Href
	publi.Type = "application/epub+zip"
	//publi.Templated = false
	l.Links["publication"] = *publi

	if _, present := config.Config.License.Links["status"]; present { // add status server to License
		status := new(license.Link) //status.Type = ??
		status.Href = config.Config.License.Links["status"]
		l.Links["status"] = *status
	}
	if _, present := l.Links["status"]; present {
		statusLink := strings.Replace(l.Links["status"].Href, "{license_id}", l.Id, 1)
		if statusLink != l.Links["status"].Href {
			status := new(license.Link)
			status.Href = statusLink
			l.Links["status"] = *status
		}
	}

	// finalize by ensuring type field is correctly set
	statu := new(license.Link)
	statu.Href = l.Links["status"].Href
	statu.Type = "application/vnd.readium.license.status.v1.0+json"
	//statu.Templated = false
	l.Links["status"] = *statu

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

	if l.Signature != nil {
		log.Println("Signature is NOT nil (it should)")
		l.Signature = nil
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
	w.Header().Set("Content-Type", "application/json")
	var page int64
	var per_page int64
	var err error
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}
	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 {
		page -= 1 //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	licenses := make([]license.LicenseReport, 0)
	//log.Println("ListAll(" + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
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
		return
	}
}

//ListLicenses returns a JSON struct with information about emitted licenses
// content-id is in url
// optional GET parameters are "page" (page number) and "per_page" (items par page)
func ListLicensesForContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var page int64
	var per_page int64
	var err error
	contentId := vars["key"]
	w.Header().Set("Content-Type", "application/json")
	//check if license exists
	_, err = s.Index().Get(contentId)
	if err == index.NotFound {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusNotFound)
		return
	} //other errors pass, but will probably reoccur
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt(r.FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}

	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 {
		page -= 1 //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	licenses := make([]license.LicenseReport, 0)
	//log.Println("List(" + contentId + "," + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
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
		return
	}

}
