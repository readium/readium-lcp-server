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

package apilcp

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
	"github.com/readium/readium-lcp-server/sign"
	"github.com/readium/readium-lcp-server/storage"
)

// GetLicense returns a license, out of a license id and partial license given as input
//
func GetLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)

	licenceID := vars["license_id"]

	// get the license from the db, by its id
	var ExistingLicense license.License
	ExistingLicense, e := s.Licenses().Get(licenceID)
	if e != nil {
		if e == license.NotFound {
			problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
		}
		return
	}
	// get the partial license given as input
	var lic license.License
	err := DecodeJSONLicense(r, &lic)

	// debug log
	//log.Println("PARTIAL LICENSE RECEIVED IN REQUEST BODY:")
	//spew.Dump(lic)

	if err != nil { // no or incorrect (json) license found in body

		log.Println("PARTIAL CONTENT:(error: " + err.Error() + ")")
		body, _ := ioutil.ReadAll(r.Body)
		log.Println("BODY=" + string(body))
		err = prepareLinks(ExistingLicense, s)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", api.ContentType_LCP_JSON)

		// must come *after* w.Header().Add()/Set(), but before w.Write()
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

		// debug only
		//log.Println("PARTIAL LICENSE FOR RESPONSE:")
		//spew.Dump(ExistingLicense)

		return
	}

	// add information to the license, sign and return it
	if lic.User.Email == "" {
		problem.Error(w, r, problem.Problem{Detail: "User information must be passed in INPUT"}, http.StatusBadRequest)
		return
	}
	ExistingLicense.User = lic.User

	if ExistingLicense.Links == nil {
		ExistingLicense.Links = license.DefaultLinksCopy()
	}

	ExistingLicense.Encryption.UserKey.Value = lic.Encryption.UserKey.Value

	err = completeLicense(&ExistingLicense, ExistingLicense.ContentId, s)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	enc.Encode(ExistingLicense)

	// debug only
	//log.Println("COMPLETE LICENSE FOR RESPONSE:")
	//spew.Dump(ExistingLicense)
}

// this function only updates the license in the database given a partial license input
func updateLicenseInDatabase(licenseID string, partialLicense license.License, s Server) (license.License, error) {

	var ExistingLicense license.License
	ExistingLicense, err := s.Licenses().Get(licenseID)
	if err != nil {
		return ExistingLicense, err
	}

	// update rights of license in database / verify validity of lic / existingLicense
	if partialLicense.Provider != "" {
		ExistingLicense.Provider = partialLicense.Provider
	}
	if partialLicense.Rights != nil {
		if partialLicense.Rights.Copy != nil {
			ExistingLicense.Rights.Copy = partialLicense.Rights.Copy
		}
		if partialLicense.Rights.Print != nil {
			ExistingLicense.Rights.Print = partialLicense.Rights.Print
		}
		if partialLicense.Rights.Start != nil {
			ExistingLicense.Rights.Start = partialLicense.Rights.Start
		}
		if partialLicense.Rights.End != nil {
			ExistingLicense.Rights.End = partialLicense.Rights.End
		}
	} else {
		ExistingLicense.Rights.Copy = nil
		ExistingLicense.Rights.Print = nil
		ExistingLicense.Rights.Start = nil
		ExistingLicense.Rights.End = nil
	}

	if partialLicense.Encryption.UserKey.Hint != "" {
		ExistingLicense.Encryption.UserKey.Hint = partialLicense.Encryption.UserKey.Hint
	}
	if partialLicense.ContentId != "" { //change content
		ExistingLicense.ContentId = partialLicense.ContentId
	}
	err = s.Licenses().Update(ExistingLicense)
	if err != nil { // no or incorrect (json) license found in body
		return ExistingLicense, err
	}
	return ExistingLicense, err
}

// UpdateLicense updates an existing license and returns the new license (lcpl)
//
func UpdateLicense(w http.ResponseWriter, r *http.Request, s Server) {

	vars := mux.Vars(r)
	licenseID := vars["license_id"]
	var lic license.License
	err := DecodeJSONLicense(r, &lic)
	if err != nil { // no or incorrect (json) license found in body
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	if lic.Id != licenseID {
		problem.Error(w, r, problem.Problem{Detail: "Different license IDs"}, http.StatusNotFound)
		return
	}
	_, err = updateLicenseInDatabase(licenseID, lic, s)
	if err != nil {
		if err == license.NotFound {
			problem.Error(w, r, problem.Problem{Detail: license.NotFound.Error()}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		}
		return
	}
	// go on and GET license io to return the updated license
	GetLicense(w, r, s)
}

// TODO: the UpdateRightsLicense function appears to be unused?
// func UpdateRightsLicense(w http.ResponseWriter, r *http.Request, s Server) {
// 	vars := mux.Vars(r)
// 	licenceId := vars["key"]
// 	// search existing license using key
// 	var ExistingLicense license.License
// 	ExistingLicense, e := s.Licenses().Get(licenceId)
// 	if e != nil {
// 		if e == license.NotFound {
// 			problem.Error(w, r, problem.Problem{Detail: license.NotFound.Error()}, http.StatusNotFound)
// 		} else {
// 			problem.Error(w, r, problem.Problem{Detail: e.Error()}, http.StatusBadRequest)
// 		}
// 		return
// 	}
// 	var lic license.License
// 	err := DecodeJSONLicense(r, &lic)
// 	if err != nil { // no or incorrect (json) license found in body
// 		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
// 		return
// 	}
// 	if lic.Id != licenceId {
// 		problem.Error(w, r, problem.Problem{Detail: "Different license IDs"}, http.StatusNotFound)
// 		return
// 	}
// 	// update rights of license in database
// 	if lic.Rights.Copy != nil {
// 		ExistingLicense.Rights.Copy = lic.Rights.Copy
// 	}
// 	if lic.Rights.Print != nil {
// 		ExistingLicense.Rights.Print = lic.Rights.Print
// 	}
// 	if lic.Rights.Start != nil {
// 		ExistingLicense.Rights.Start = lic.Rights.Start
// 	}
// 	if lic.Rights.End != nil {
// 		ExistingLicense.Rights.End = lic.Rights.End
// 	}
// 	err = s.Licenses().UpdateRights(ExistingLicense)
// 	if err != nil { // no or incorrect (json) license found in body
// 		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
// 		return
// 	}
// 	// go on to GET license io to return the existing license
// 	GetLicense(w, r, s)
// }

// GenerateLicense generates and returns a new license,
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateLicense(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var lic license.License

	err := DecodeJSONLicense(r, &lic)

	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	contentID := vars["content_id"]
	lic.ContentId = ""
	err = completeLicense(&lic, contentID, s)

	if err != nil {
		if err == storage.ErrNotFound || err == index.NotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}

	err = s.Licenses().Add(lic)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", api.ContentType_LCP_JSON)
	w.Header().Add("Content-Disposition", `attachment; filename="license.lcpl"`)

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	enc.Encode(lic)
}

// GenerateProtectedPublication generates and returns a protected publication
// for a given license identified by its id
// or
// for a given content identified by its id
// plus a partial license given as input
//
func GenerateProtectedPublication(w http.ResponseWriter, r *http.Request, s Server) {
	var partialLicense license.License
	var newLicense license.License
	err := DecodeJSONLicense(r, &partialLicense)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	contentID := vars["content_id"] // may be empty
	licenseID := vars["license_id"] // may be empty

	if licenseID != "" { // POST /{license_id}/publication
		//license update, regenerate publication, maybe only get from db ?
		newLicense, err = updateLicenseInDatabase(licenseID, partialLicense, s)
		if err != nil {
			if err == license.NotFound {
				problem.Error(w, r, problem.Problem{Detail: license.NotFound.Error()}, http.StatusNotFound)
			} else {
				problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			}
			return
		}
		newLicense.User = partialLicense.User //pass user information in updated license
		newLicense.Encryption.UserKey.Value = partialLicense.Encryption.UserKey.Value
		newLicense.Encryption.UserKey.ClearValue = partialLicense.Encryption.UserKey.ClearValue

		// contentID is not set, get it from the license
		contentID = newLicense.ContentId
		err = completeLicense(&newLicense, contentID, s)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}

	} else { //	 POST //{content_id}/publication[s]
		//new license, generate publication
		newLicense = partialLicense
		newLicense.ContentId = ""
		err = completeLicense(&newLicense, contentID, s)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}
		err = s.Licenses().Add(newLicense)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
			return
		}
		licenseID = newLicense.Id
	}

	// get the publication, add the license to the publication
	if contentID == "" {
		problem.Error(w, r, problem.Problem{Detail: "No content id", Instance: contentID}, http.StatusBadRequest)
		return
	}

	epubFile, err := s.Store().Get(contentID)
	if err != nil {
		if err == storage.ErrNotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
			return
		}
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	content, err := s.Index().Get(contentID)
	if err != nil {
		if err == index.NotFound {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusNotFound)
		} else {
			problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		}
		return
	}
	var b bytes.Buffer
	contents, err := epubFile.Contents()
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}

	io.Copy(&b, contents)
	zr, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error(), Instance: contentID}, http.StatusInternalServerError)
		return
	}
	//add license to publication
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(newLicense)
	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add(epub.LicenseFile, &buf2, uint64(buf2.Len()))

	//set HTTP headers
	w.Header().Add("Content-Type", epub.ContentType_EPUB)
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, content.Location))
	w.Header().Add("X-Lcp-License", newLicense.Id)
	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(http.StatusCreated)
	// write HTTP body
	ep.Write(w)
}

// DecodeJSONLicense decodes a license formatted in json and returns a license object
//
func DecodeJSONLicense(r *http.Request, lic *license.License) error {
	var dec *json.Decoder

	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_FORM_URL_ENCODED {
		buf := bytes.NewBufferString(r.PostFormValue("data"))
		dec = json.NewDecoder(buf)
	} else {
		dec = json.NewDecoder(r.Body)
	}

	err := dec.Decode(&lic)

	return err
}

// completeLicense generates a complete license out of a partial license
//
func completeLicense(l *license.License, contentID string, s Server) error {
	c, err := s.Index().Get(contentID)
	if err != nil {
		return err
	}

	isNewLicense := l.ContentId == ""
	if isNewLicense {
		license.Prepare(l)
		l.ContentId = contentID
	} else {
		l.Signature = nil // empty signature fields, needs to be recalculated
	}
	links := new([]license.Link)

	// verify that mandatory hint link is present in the configuration
	if value, present := license.DefaultLinks["hint"]; present {
		hint := license.Link{Href: value, Rel: "hint"}
		*links = append(*links, hint)
	} else {
		return errors.New("No hint link present in the config file")
	}

	// verify that mandatory publication link is present in the configuration
	// replace the publication_id template by the actual value
	if value, present := license.DefaultLinks["publication"]; present {
		if !strings.Contains(value, "{publication_id}") {
			return errors.New("Missing {publication_id} template in publication link in config file")
		}
		// replace {publication_id} in template link
		publicationLink := strings.Replace(value, "{publication_id}", c.Id, 1)
		publication := license.Link{Href: publicationLink, Rel: "publication", Type: epub.ContentType_EPUB, Size: c.Length, Title: c.Location, Checksum: c.Sha256}
		*links = append(*links, publication)
	} else {
		return errors.New("Missing publication link in config file")
	}

	// verify that mandatory status link is present in the configuration
	// note that the status link is made mandatory in the production lcp ecosystem
	// replace the license_id template by the actual value
	if value, present := config.Config.License.Links["status"]; present { // add status server to License
		if !strings.Contains(value, "{license_id}") {
			return errors.New("Missing {license_id} template in status link in config file")
		}
		statusLink := strings.Replace(value, "{license_id}", l.Id, 1)

		status := license.Link{Href: statusLink, Rel: "status", Type: api.ContentType_LSD_JSON} //status.Type = ??
		*links = append(*links, status)
	} else {
		return errors.New("Missing status link in config file")
	}

	l.Links = *links

	// Generate user key
	encryptionKey := license.GenerateUserKey(l.Encryption.UserKey)

	// Empty UserKey clear value to avoid clear passphrase to be sent to the
	// final user
	l.Encryption.UserKey.ClearValue = ""

	// Encrypt content key with user key
	encrypterContentKey := crypto.NewAESEncrypter_CONTENT_KEY()

	l.Encryption.ContentKey.Algorithm = encrypterContentKey.Signature()
	l.Encryption.ContentKey.Value = encryptKey(encrypterContentKey, c.EncryptionKey, encryptionKey[:])
	l.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"

	encrypterFields := crypto.NewAESEncrypter_FIELDS()

	err = encryptFields(encrypterFields, l, encryptionKey[:])
	if err != nil {
		return err
	}

	encrypterUserKeyCheck := crypto.NewAESEncrypter_USER_KEY_CHECK()

	err = buildKeyCheck(encrypterUserKeyCheck, l, encryptionKey[:])
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

func buildKeyCheck(encrypter crypto.Encrypter, l *license.License, key []byte) error {
	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(l.Id), &out)
	if err != nil {
		return err
	}
	l.Encryption.UserKey.Check = out.Bytes()
	return nil
}

func encryptFields(encrypter crypto.Encrypter, l *license.License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(&l.User, toEncrypt)
		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
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

func encryptKey(encrypter crypto.Encrypter, key []byte, kek []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(key)
	encrypter.Encrypt(kek[:], in, &out)
	return out.Bytes()
}

//ListLicenses returns a JSON struct with information about emitted licenses
// optional GET parameters are "page" (page number) and "per_page" (items par page)
func ListLicenses(w http.ResponseWriter, r *http.Request, s Server) {
	var page int64
	var per_page int64
	var err error
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}
	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 { //pagenum starting at 0 in code, but user interface starting at 1
		page--
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
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
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

// ListLicensesForContent lists all licenses associated with a given content
// the content id is passed in the calling url
// optional GET parameters are "page" (page number) and "per_page" (items par page)
func ListLicensesForContent(w http.ResponseWriter, r *http.Request, s Server) {
	vars := mux.Vars(r)
	var page int64
	var per_page int64
	var err error
	contentId := vars["content_id"]
	//check if license exists
	_, err = s.Index().Get(contentId)
	if err == index.NotFound {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		return
	} //other errors pass, but will probably reoccur
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt(r.FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}

	if r.FormValue("per_page") != "" {
		per_page, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		per_page = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
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
	w.Header().Set("Content-Type", api.ContentType_JSON)
	enc := json.NewEncoder(w)
	err = enc.Encode(licenses)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}

}

func prepareLinks(license license.License, s Server) error {
	for i := 0; i < len(license.Links); i++ {
		if license.Links[i].Rel == "publication" {
			item, err := s.Index().Get(license.ContentId)
			if err != nil {
				return err
			}
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{publication_id}", license.ContentId, 1)
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{publication_loc}", item.Location, 1)
		}

		if license.Links[i].Rel == "status" {
			license.Links[i].Href = strings.Replace(license.Links[i].Href, "{license_id}", license.Id, 1)
		}
	}
	return nil
}
