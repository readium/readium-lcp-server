/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package lcpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/sign"
	"github.com/readium/readium-lcp-server/store"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

func writeRequestFileToTemp(r io.Reader) (int64, *os.File, error) {
	dir := os.TempDir()
	file, err := ioutil.TempFile(dir, "readium-lcp")
	if err != nil {
		return 0, file, err
	}

	n, err := io.Copy(file, r)

	// Rewind to the beginning of the file
	file.Seek(0, 0)

	return n, file, err
}

func cleanupTemp(f *os.File) {
	if f == nil {
		return
	}
	f.Close()
	os.Remove(f.Name())
}

// notifyLSDServer informs the License Status Server of the creation of a new license
// and saves the result of the http request in the DB (using *LicenseRepository)
//
func notifyLSDServer(payload *store.License, server api.IServer) {
	if server.Config().LsdServer.PublicBaseUrl == "" {
		// can't call : url is empty
		return
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		server.LogError("Error Notify LsdServer of new License (" + payload.Id + ") Marshaling error : " + err.Error())
	}

	req, err := http.NewRequest("PUT", server.Config().LsdServer.PublicBaseUrl+"/licenses", bytes.NewReader(jsonPayload))
	if err != nil {
		return
	}

	// set credentials on lsd request
	notifyAuth := server.Config().LsdNotifyAuth
	if notifyAuth.Username != "" {
		req.SetBasicAuth(notifyAuth.Username, notifyAuth.Password)
	}
	req.Header.Add(api.HdrContentType, api.ContentTypeLcpJson)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		server.LogError("Error Notify LsdServer of new License (" + payload.Id + "):" + err.Error())
		err = server.Store().License().UpdateLsdStatus(payload.Id, -1)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		server.LogError("Error Notify LsdServer of new License (read body error : %v)", err)
	}
	_ = server.Store().License().UpdateLsdStatus(payload.Id, int32(resp.StatusCode))
	// message to the console
	server.LogInfo("Notify LsdServer of a new License with id %q http-status %d response %v", payload.Id, resp.StatusCode, body)

}

func getField(info *store.User, field string) reflect.Value {
	value := reflect.ValueOf(info).Elem()
	return value.FieldByName(strings.Title(field))
}

// buildKeyCheck
// encrypt the license id with the key used for encrypting content
//
func buildKeyCheck(licenseID string, encrypter crypto.Encrypter, key []byte) ([]byte, error) {
	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(licenseID), &out)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func encryptKey(encrypter crypto.Encrypter, fromKey []byte, key []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(fromKey)
	encrypter.Encrypt(key[:], in, &out)
	return out.Bytes()
}

func encryptFields(encrypter crypto.Encrypter, l *store.License, key []byte) error {
	for _, toEncrypt := range l.User.Encrypted {
		var out bytes.Buffer
		field := getField(l.User, toEncrypt)
		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

// EncryptLicenseFields sets the content key, encrypted user info and key check
//
func EncryptLicenseFields(license *store.License, content *store.Content) error {

	// generate the user key
	encryptionKey := []byte(license.Encryption.UserKey.Value)

	// empty the passphrase hash to avoid sending it back to the user
	license.Encryption.UserKey.Value = ""

	// encrypt the content key with the user key
	encrypterContentKey := crypto.NewAESEncrypterContentKey()
	license.Encryption.ContentKey.Algorithm = encrypterContentKey.Signature()
	license.Encryption.ContentKey.Value = encryptKey(encrypterContentKey, content.EncryptionKey, encryptionKey[:])

	// encrypt the user info fields
	encrypterFields := crypto.NewAESEncrypterFields()
	err := encryptFields(encrypterFields, license, encryptionKey[:])
	if err != nil {
		return err
	}

	// build the key check
	encrypterUserKeyCheck := crypto.NewAESEncrypterUserKeyCheck()
	chk, err := buildKeyCheck(license.Id, encrypterUserKeyCheck, encryptionKey[:])
	if err != nil {
		return err
	}
	license.Encryption.UserKey.Check = string(chk)
	return nil
}

// build a license, common to get and generate license, get and generate licensed publication
//
func buildLicense(license *store.License, server api.IServer) error {

	// set the LCP profile
	// possible profiles are basic and 1.0
	if server.Config().Profile == "1.0" {
		license.Encryption.Profile = store.V1Profile
	} else {
		license.Encryption.Profile = store.BasicProfile
	}

	// get content info from the db
	content, err := server.Store().Content().Get(license.ContentId)
	if err != nil {
		server.LogError("No content with id %v %v", license.ContentId, err)
		return err
	}
	server.LogInfo("setting license links.")
	// set links
	err = server.SetLicenseLinks(license, content)
	if err != nil {
		return err
	}
	server.LogInfo("Encrypting fields.")
	// encrypt the content key, user fieds, set the key check
	err = EncryptLicenseFields(license, content)
	if err != nil {
		return err
	}
	server.LogInfo("Signing license.")
	// sign the license
	err = SignLicense(license, server.Certificate())
	if err != nil {
		return err
	}
	return nil
}

// SignLicense signs a license using the server certificate
//
func SignLicense(license *store.License, cert *tls.Certificate) error {
	sig, err := sign.NewSigner(cert)
	if err != nil {
		return err
	}
	res, err := sig.Sign(license)
	if err != nil {
		return err
	}
	license.Signature = &res

	return nil
}

// build a licensed publication, common to get and generate licensed publication
//
func buildLicencedPublication(license *store.License, server api.IServer) (*epub.Epub, error) {
	// get the epub content info from the bd
	epubFile, err := server.Storage().Get(license.ContentId)
	if err != nil {
		return nil, err
	}
	// get the epub content
	epubContent, err1 := epubFile.Contents()
	if err1 != nil {
		return nil, err1
	}
	var b bytes.Buffer
	// copy the epub content to a buffer
	io.Copy(&b, epubContent)
	// create a zip reader
	zr, err2 := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err2 != nil {
		return nil, err2
	}
	ep, err3 := epub.Read(zr)
	if err3 != nil {
		return nil, err3
	}
	// add the license to publication
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	enc.Encode(license)
	// write the buffer in the zip, and suppress the trailing newline
	// FIXME: check that the newline is not present anymore
	// FIXME/ try to optimize with buf.ReadBytes(byte('\n')) instead of creating a new buffer.
	var buf2 bytes.Buffer
	buf2.Write(bytes.TrimRight(buf.Bytes(), "\n"))
	ep.Add(epub.LicenseFile, &buf2, uint64(buf2.Len()))
	return &ep, err
}
