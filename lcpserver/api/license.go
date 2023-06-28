// Copyright 2017 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package apilcp

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/license"
)

// ErrBadInputLicense sets an error message returned to the caller and indicating the user-supplied
// input license is invalid for the called function
type ErrBadLicenseInput struct {
	msg string
}

func (e ErrBadLicenseInput) Error() string {
	return fmt.Sprintf("bad license input: %s", e.msg)
}

// MandatoryInfoMissingMsg sets an error message returned to the caller
var MandatoryInfoMissingMsg = "mandatory info missing in the input body"

// BadHexValueMsg sets an error message returned to the caller
var BadHexValueMsg = "erroneous user_key.hex_value can't be decoded"

// BadUserKeyValueMsg sets an error message returned to the caller
var BadUserKeyValueMsg = "erroneous user_key.value, can't be decoded"

// checkGetLicenseInput: if we generate or get a license, check mandatory information in the input body
// and compute request parameters
func checkGetLicenseInput(l *license.License) error {

	// the user hint is mandatory
	if l.Encryption.UserKey.Hint == "" {
		log.Println("User hint is missing")
		return ErrBadLicenseInput{MandatoryInfoMissingMsg}
	}
	// Value or HexValue are mandatory
	// HexValue (hex encoded passphrase hash) takes precedence over Value (kept for backward compatibility)
	// Value is computed from HexValue if set
	if l.Encryption.UserKey.HexValue != "" {
		// compute a byte array from a string
		value, err := hex.DecodeString(l.Encryption.UserKey.HexValue)
		if err != nil {
			return ErrBadLicenseInput{BadHexValueMsg}
		}
		l.Encryption.UserKey.Value = value
	} else if l.Encryption.UserKey.Value == nil {
		log.Println("User hashed passphrase is missing")
		return ErrBadLicenseInput{MandatoryInfoMissingMsg}
	}
	// check the size of Value (32 bytes), to avoid weird errors in the crypto code
	if len(l.Encryption.UserKey.Value) != 32 {
		return ErrBadLicenseInput{BadUserKeyValueMsg}
	}

	return nil
}

// checkGenerateLicenseInput: if we generate a license, check mandatory information in the input body
func checkGenerateLicenseInput(l *license.License) error {

	if l.User.ID == "" {
		log.Println("User identification is missing")
		return ErrBadLicenseInput{MandatoryInfoMissingMsg}
	}
	// check user hint, passphrase hash and hash algorithm
	err := checkGetLicenseInput(l)
	if err != nil {
		return ErrBadLicenseInput{err.Error()}
	}
	return nil
}

// get license, copy useful data from licIn to LicOut
func copyInputToLicense(licIn *license.License, licOut *license.License) {

	// copy the user hint and hashed passphrase
	licOut.Encryption.UserKey.Hint = licIn.Encryption.UserKey.Hint
	licOut.Encryption.UserKey.Value = licIn.Encryption.UserKey.Value
	licOut.Encryption.UserKey.HexValue = licIn.Encryption.UserKey.HexValue

	// copy optional user information
	licOut.User.Email = licIn.User.Email
	licOut.User.Name = licIn.User.Name
	licOut.User.Encrypted = licIn.User.Encrypted
	licOut.Links = licIn.Links
}

// normalize the start and end date, UTC, no milliseconds
func setRights(lic *license.License) {

	// a rights object is needed before adding a record to the db
	if lic.Rights == nil {
		lic.Rights = new(license.UserRights)
	}
	if lic.Rights.Start != nil {
		start := lic.Rights.Start.UTC().Truncate(time.Second)
		lic.Rights.Start = &start
	}
	if lic.Rights.End != nil {
		end := lic.Rights.End.UTC().Truncate(time.Second)
		lic.Rights.End = &end
	}
}

// build a license, common to get and generate license, get and generate licensed publication
func buildLicense(lic *license.License, s Server, updatefix bool) error {

	// set the LCP profile
	license.SetLicenseProfile(lic)

	// force the algorithm to the one defined by the basic and 1.0 profiles
	lic.Encryption.UserKey.Algorithm = "http://www.w3.org/2001/04/xmlenc#sha256"

	// get content info from the db
	content, err := s.Index().Get(lic.ContentID)
	if err != nil {
		log.Println("No content with id", lic.ContentID)
		return err
	}

	// set links
	err = license.SetLicenseLinks(lic, content)
	if err != nil {
		return err
	}
	// encrypt the content key, user fieds, set the key check
	err = license.EncryptLicenseFields(lic, content)
	if err != nil {
		return err
	}

	// fix an issue with clients which test that the date of last update of the license
	// is after the date of creation of the X509 certificate.
	// Because of this, when a cert is replaced, fresh licenses are not accepted by such clients
	// when they have been created / updated before the cert update.
	if updatefix && config.Config.LcpServer.CertDate != "" {
		certDate, err := time.Parse("2006-01-02", config.Config.LcpServer.CertDate)
		if err != nil {
			return err
		}
		if lic.Issued.Before(certDate) && (lic.Updated == nil || lic.Updated.Before(certDate)) {
			lic.Updated = &certDate
		}
	}

	// sign the license
	err = license.SignLicense(lic, s.Certificate())
	if err != nil {
		return err
	}
	return nil
}

// copyZipFiles copies every file from one zip archive to another
func copyZipFiles(out *zip.Writer, in *zip.Reader) error {

	for _, file := range in.File {
		newFile, err := out.CreateHeader(&file.FileHeader)
		if err != nil {
			return err
		}

		r, err := file.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(newFile, r)
		r.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// isWebPub checks the presence of a REadium manifest is a zip package
func isWebPub(in *zip.Reader) bool {

	for _, f := range in.File {
		if f.Name == "manifest.json" {
			return true
		}
	}

	return false
}

// BuildLicensedPublication builds a licensed publication, common to get and generate licensed publication
func BuildLicensedPublication(lic *license.License, s Server) (buf bytes.Buffer, contentLocation string, err error) {

	// get content info from the db
	item, err := s.Store().Get(lic.ContentID)
	if err != nil {
		return
	}
	// read the content into a buffer
	contents, err := item.Contents()
	if err != nil {
		return buf, contentLocation, err
	}
	b, err := ioutil.ReadAll(contents)
	if err != nil {
		return buf, contentLocation, err
	}
	// create a zip reader
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return buf, contentLocation, err
	}

	zipWriter := zip.NewWriter(&buf)
	err = copyZipFiles(zipWriter, zr)
	if err != nil {
		return buf, contentLocation, err
	}

	// Encode the license to JSON, remove the trailing newline
	// write the buffer in the zip
	licenseBytes, err := json.Marshal(lic)
	if err != nil {
		return buf, contentLocation, err
	}

	licenseBytes = bytes.TrimRight(licenseBytes, "\n")

	licenseLocation := epub.LicenseFile
	if isWebPub(zr) {
		licenseLocation = "license.lcpl"
	}

	licenseWriter, err := zipWriter.Create(licenseLocation)
	if err != nil {
		return buf, contentLocation, err
	}

	_, err = licenseWriter.Write(licenseBytes)
	if err != nil {
		return
	}

	// get the content location to fill an http header
	// FIXME: redundant as the content location has been set in a link (publication)
	content, err := s.Index().Get(lic.ContentID)
	if err != nil {
		return buf, contentLocation, err
	}
	contentLocation = content.Location

	return buf, contentLocation, zipWriter.Close()
}

func GetLicense(licenseID string, licIn *license.License, s Server) (*license.License, error) {
	// an input body was sent with the request:
	// check mandatory information in the partial license
	err := checkGetLicenseInput(licIn)
	if err != nil {
		return nil, ErrBadLicenseInput{err.Error()}
	}

	// initialize the license from the info stored in the db.
	licOut, err := s.Licenses().Get(licenseID)
	if err != nil {
		return nil, err
	}

	// copy useful data from licIn to LicOut
	copyInputToLicense(licIn, &licOut)

	err = buildLicense(&licOut, s, true)
	if err != nil {
		return nil, err
	}

	return &licOut, nil
}

func GenerateLicense(contentID string, lic *license.License, s Server) error {
	// check mandatory information in the input body
	err := checkGenerateLicenseInput(lic)
	if err != nil {
		return ErrBadLicenseInput{err.Error()}
	}
	// init the license with an id and issue date
	license.Initialize(contentID, lic)

	// normalize the start and end date, UTC, no milliseconds
	setRights(lic)

	// build the license
	err = buildLicense(lic, s, false)
	if err != nil {
		return err
	}

	// store the license in the db
	err = s.Licenses().Add(*lic)
	if err != nil {
		return err
	}
	log.Println("New License:", lic.ID, ". Content:", contentID, "User:", lic.User.ID)

	// notify the lsd server of the creation of the license.
	// this is an asynchronous call.
	go notifyLsdServer(*lic, s)

	return nil
}

func UpdateLicense(licenseID string, licIn *license.License, s Server) error {
	// initialize the license from the info stored in the db.
	var licOut license.License
	licOut, err := s.Licenses().Get(licenseID)
	if err != nil {
		return err
	}

	// update licOut using information found in licIn
	if licIn.User.ID != "" {
		log.Println("new user id: ", licIn.User.ID)
		licOut.User.ID = licIn.User.ID
	}
	if licIn.Provider != "" {
		log.Println("new provider: ", licIn.Provider)
		licOut.Provider = licIn.Provider
	}
	if licIn.ContentID != "" {
		log.Println("new content id: ", licIn.ContentID)
		licOut.ContentID = licIn.ContentID
	}
	if licIn.Rights.Print != nil {
		log.Println("new right, print: ", *licIn.Rights.Print)
		licOut.Rights.Print = licIn.Rights.Print
	}
	if licIn.Rights.Copy != nil {
		log.Println("new right, copy: ", *licIn.Rights.Copy)
		licOut.Rights.Copy = licIn.Rights.Copy
	}
	if licIn.Rights.Start != nil {
		log.Println("new right, start: ", *licIn.Rights.Start)
		licOut.Rights.Start = licIn.Rights.Start
	}
	if licIn.Rights.End != nil {
		log.Println("new right, end: ", *licIn.Rights.End)
		licOut.Rights.End = licIn.Rights.End
	}
	// update the license in the database
	err = s.Licenses().Update(licOut)
	if err != nil {
		return err
	}
	return nil
}

// DecodeJSONLicense decodes a license formatted in json and returns a license object
func DecodeJSONLicense(dec *json.Decoder, lic *license.License) error {
	err := dec.Decode(&lic)

	if err != nil && err.Error() != "EOF" {
		log.Print("Decode license: invalid json structure")
	}
	return err
}

// notifyLsdServer informs the License Status Server of the creation of a new license
// and saves the result of the http request in the DB (using *Store)
func notifyLsdServer(l license.License, s Server) {

	if config.Config.LsdServer.PublicBaseUrl != "" {
		var lsdClient = &http.Client{
			Timeout: time.Second * 10,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_ = json.NewEncoder(pw).Encode(l)
			pw.Close() // signal end writing
		}()
		req, err := http.NewRequest("PUT", config.Config.LsdServer.PublicBaseUrl+"/licenses", pr)
		if err != nil {
			return
		}
		// set credentials on lsd request
		notifyAuth := config.Config.LsdNotifyAuth
		if notifyAuth.Username != "" {
			req.SetBasicAuth(notifyAuth.Username, notifyAuth.Password)
		}

		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)

		response, err := lsdClient.Do(req)
		if err != nil {
			log.Println("Error Notify LsdServer of new License (" + l.ID + "):" + err.Error())
			_ = s.Licenses().UpdateLsdStatus(l.ID, -1)
		} else {
			defer req.Body.Close()
			_ = s.Licenses().UpdateLsdStatus(l.ID, int32(response.StatusCode))
		}
	}
}
