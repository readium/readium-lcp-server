// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	uuid "github.com/satori/go.uuid"
)

// notification of newly added content (Publication)
func notifyLcpServer(lcpService, contentid string, lcpPublication apilcp.LcpPublication, username string, password string) error {

	//exchange encryption key with lcp service/content/<id>,
	//Payload:
	//  content-id: unique id for the content
	//  content-encryption-key: encryption key used for the content
	//  protected-content-location: full path of the encrypted file
	//  protected-content-length: content length in bytes
	//  protected-content-sha256: content sha
	//  protected-content-disposition: encrypted file name
	//  protected-content-type: encrypted file content type
	//fmt.Printf("lcpsv = %s\n", *lcpsv)
	var urlBuffer bytes.Buffer
	urlBuffer.WriteString(lcpService)
	urlBuffer.WriteString("/contents/")
	urlBuffer.WriteString(contentid)

	jsonBody, err := json.Marshal(lcpPublication)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", urlBuffer.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if (resp.StatusCode != 302) && (resp.StatusCode/100) != 2 { //302=found or 20x reply = OK
		return fmt.Errorf("lcp server error %d", resp.StatusCode)
	}

	return nil
}

// getBufferFromInputResource reads and returns in a buffer
// the content of a file on the local filesystem
// or the content of a remote resource if the scheme is http:// or https://
func getBufferFromInputResource(inputPath string) ([]byte, error) {

	url, err := url.Parse(inputPath)
	if err != nil {
		return nil, errors.New("Error parsing input file")
	}
	if url.Scheme == "http" || url.Scheme == "https" {
		res, err := http.Get(inputPath)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		return ioutil.ReadAll(res.Body)
	} else if url.Scheme == "ftp" {
		return nil, errors.New("ftp not supported yet")

	} else {
		return ioutil.ReadFile(inputPath)
	}
}

func showHelpAndExit() {

	log.Println("lcpencrypt protects an epub/pdf file for usage in an lcp environment")
	log.Println("-input        source epub/pdf/lpf file locator (file system or http GET)")
	log.Println("[-profile]    encryption profile")
	log.Println("[-contentid]  optional content identifier, if omitted a new one will be generated")
	log.Println("[-output]     optional target location for protected content (file system or http PUT)")
	log.Println("[-lcpsv]      optional http endpoint for the License server")
	log.Println("[-login]      login ( needed for License server) ")
	log.Println("[-password]   password ( needed for License server)")
	log.Println("[-help] :     help information")
	os.Exit(0)
	return
}

func exitWithError(lcpPublication apilcp.LcpPublication, err error, errorlevel int) {

	os.Stdout.WriteString(lcpPublication.ErrorMessage + "; level " + strconv.Itoa(errorlevel))
	os.Stdout.WriteString("\n")
	if err != nil {
		os.Stdout.WriteString(err.Error())
	}
	os.Stdout.WriteString("\n")
	/* kept for future debug
	jsonBody, err := json.MarshalIndent(lcpPublication, " ", "  ")
	if err != nil {
		os.Stdout.WriteString("Error creating json lcpPublication\n")
		os.Exit(errorlevel)
	}
	os.Stdout.Write(jsonBody)
	os.Stdout.WriteString("\n")
	*/
	os.Exit(errorlevel)
}

func getChecksum(file *os.File) string {

	hasher := sha256.New()
	//s, err := ioutil.ReadFile(filename) // filename as string
	//hasher.Write(s)
	// Note: this alternative code should save memory (no s buffer); found in lcpencrypt.go
	file.Seek(0, 0)
	if _, err := io.Copy(hasher, file); err != nil {
		return ""
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func outputExtension(sourceExt string) string {

	var targetExt string
	switch sourceExt {
	case ".epub":
		targetExt = ".epub"
	case ".pdf":
		targetExt = ".lcpdf"
	case ".lpf":
		// short term solution. We'll need to inspect the manifest and check conformsTo,
		// to be certain this is an audiobook (vs another profile of Web Publication)
		targetExt = ".audiobook"
	}
	return targetExt
}

// buildEncryptedRWPP builds an encrypted Readium package out of an un-encrypted one
// FIXME: it cannot be used for EPUB as long as Do() and Process() are not merged
func buildEncryptedRWPP(pub *apilcp.LcpPublication, inputPath string, encrypter crypto.Encrypter, lcpProfile license.EncryptionProfile) error {

	// create a reader on the un-encrypted readium package
	reader, err := pack.OpenRWPP(inputPath)
	if err != nil {
		pub.ErrorMessage = "Error opening the temp package"
		return err
	}
	// create the encrypted package file
	outputFile, err := os.Create(pub.Output)
	if err != nil {
		pub.ErrorMessage = "Error creating the output package"
		return err
	}
	defer outputFile.Close()
	// create a writer on the encrypted package
	writer, err := reader.NewWriter(outputFile)
	if err != nil {
		pub.ErrorMessage = "Error opening the output package"
		return err
	}
	// encrypt resources from the input package, return the encryption key
	encryptionKey, err := pack.Process(lcpProfile, encrypter, reader, writer)
	if err != nil {
		pub.ErrorMessage = "Error encrypting the publication"
		return err
	}
	pub.ContentKey = encryptionKey
	// calculate the output file size and checksum
	stats, err := outputFile.Stat()
	if err == nil && (stats.Size() > 0) {
		filesize := stats.Size()
		pub.Size = &filesize
		cs := getChecksum(outputFile)
		pub.Checksum = &cs
	}
	if stats.Size() == 0 {
		pub.ErrorMessage = "Empty output file"
		return err
	}
	return nil
}

// processEPUB encrypts resources in an EPUB
func processEPUB(pub *apilcp.LcpPublication, inputPath string, encrypter crypto.Encrypter) error {

	pub.ContentType = epub.ContentType_EPUB

	// read the content of the input resource, *keep it in memory*
	buf, err := getBufferFromInputResource(inputPath)
	if err != nil {
		pub.ErrorMessage = "Error opening input file, for more information type 'lcpencrypt -help' "
		return err
	}
	// create a zip reader from the zipped data in the buffer
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		pub.ErrorMessage = "Error opening the zip file"
		return err
	}
	// generate an EPUB object
	epub, err := epub.Read(zr)
	if err != nil {
		pub.ErrorMessage = "Error reading the epub content"
		return err
	}
	// create the output file
	outputFile, err := os.Create(pub.Output)
	if err != nil {
		pub.ErrorMessage = "Error writing output file"
		return err
	}
	// will close the output file
	defer outputFile.Close()
	// encrypt the content of the publication,
	// write them into the output file
	_, encryptionKey, err := pack.Do(encrypter, epub, outputFile)
	if err != nil {
		pub.ErrorMessage = "Error encrypting the EPUB content"
		return err
	}
	pub.ContentKey = encryptionKey
	// calculate the output file size and checksum
	stats, err := outputFile.Stat()
	if err == nil && (stats.Size() > 0) {
		filesize := stats.Size()
		pub.Size = &filesize
		cs := getChecksum(outputFile)
		pub.Checksum = &cs
	}
	if stats.Size() == 0 {
		pub.ErrorMessage = "Empty output file"
		return err
	}
	return nil
}

// processPDF wraps an encrypted PDF file inside a Readium package
func processPDF(pub *apilcp.LcpPublication, inputPath string, encrypter crypto.Encrypter, lcpProfile license.EncryptionProfile) error {

	pub.ContentType = "application/pdf+lcp"

	// generate a temp Readium Package (rwpp) which embeds the PDF file; its title is the PDF file name
	tmpPackagePath := pub.Output + ".tmp"
	err := pack.BuildRWPPFromPDF(filepath.Base(inputPath), inputPath, tmpPackagePath)
	if err != nil {
		pub.ErrorMessage = "Error building Web Publication package from PDF"
		return err
	}
	defer os.Remove(tmpPackagePath)

	// build an encrypted package
	err = buildEncryptedRWPP(pub, tmpPackagePath, encrypter, lcpProfile)
	return err
}

// processLPF transforms a W3C LPF file into a Readium Package and encrypts its resources
func processLPF(pub *apilcp.LcpPublication, inputPath string, encrypter crypto.Encrypter, lcpProfile license.EncryptionProfile, outputExt string) error {

	switch outputExt {
	case ".audiobook":
		pub.ContentType = "application/audiobook+lcp"
	case ".divina":
		pub.ContentType = "application/divina+lcp"
	}

	// generate a tmp Readium Package (rwpp) out of W3C Package (lpf)
	tmpPackagePath := pub.Output + ".webpub"
	err := pack.BuildRWPPFromLPF(inputPath, tmpPackagePath)
	// will remove the tmp file even if an error is returned
	defer os.Remove(tmpPackagePath)
	// process error
	if err != nil {
		pub.ErrorMessage = "Error building RWPP from LPF"
		return err
	}

	// build an encrypted package
	err = buildEncryptedRWPP(pub, tmpPackagePath, encrypter, lcpProfile)
	return err
}

func main() {
	var err error
	var pub apilcp.LcpPublication
	var inputPath = flag.String("input", "", "source epub/pdf/lpf file locator (file system or http GET)")
	var contentid = flag.String("contentid", "", "optional content identifier; if omitted a new one is generated")
	var outputFilename = flag.String("output", "", "optional target location for the encrypted content (file system or http PUT)")
	var lcpsv = flag.String("lcpsv", "", "optional http endpoint of the License server (adds content)")
	var username = flag.String("login", "", "login (License server)")
	var password = flag.String("password", "", "password (License server)")
	var profile = flag.String("profile", "basic", "LCP Profile to use for encryption: 'basic' or 'v1'")

	var help = flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}
	if *help || *inputPath == "" {
		showHelpAndExit()
	}

	if *lcpsv != "" && (*username == "" || *password == "") {
		pub.ErrorMessage = "incorrect parameters, lcpsv needs a login and password, for more information type 'lcpencrypt -help' "
		exitWithError(pub, nil, 10)
	}

	if *contentid == "" { // contentID not set -> generate a new one
		uid, err := uuid.NewV4()
		if err != nil {
			exitWithError(pub, err, 20)
		}
		*contentid = uid.String()
	}
	pub.ContentID = *contentid

	// if the output file name not set,
	// then <content-id>.epub|lcpdf is created in the working directory
	inputExt := filepath.Ext(*inputPath)
	var basefilename string
	var outputExt string
	if *outputFilename == "" {
		workingDir, _ := os.Getwd()
		outputExt = outputExtension(inputExt)
		*outputFilename = strings.Join([]string{workingDir, string(os.PathSeparator), *contentid, outputExt}, "")
		basefilename = filepath.Base(*inputPath)
	} else {
		outputExt = filepath.Ext(*outputFilename)
		basefilename = filepath.Base(*outputFilename)
	}
	pub.ContentDisposition = &basefilename
	// reminder: the output path must be accessible from the license server
	pub.Output = *outputFilename

	var lcpProfile license.EncryptionProfile
	if *profile == "v1" {
		lcpProfile = license.V1Profile
	} else { // covers missing parameter
		lcpProfile = license.BasicProfile
	}

	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()

	// select the encryption process
	if inputExt == ".epub" {
		err := processEPUB(&pub, *inputPath, encrypter)
		if err != nil {
			exitWithError(pub, err, 30)
		}
	} else if inputExt == ".pdf" {
		err := processPDF(&pub, *inputPath, encrypter, lcpProfile)
		if err != nil {
			exitWithError(pub, err, 40)
		}
	} else if inputExt == ".lpf" {
		err := processLPF(&pub, *inputPath, encrypter, lcpProfile, outputExt)
		if err != nil {
			exitWithError(pub, err, 50)
		}
	}

	// notify the LCP Server
	if *lcpsv != "" {
		err = notifyLcpServer(*lcpsv, *contentid, pub, *username, *password)
		if err != nil {
			pub.ErrorMessage = "Error notifying the License Server"
			exitWithError(pub, err, 60)
		} else {
			os.Stdout.WriteString("License Server was notified\n")
		}
	}

	// write a json message to stdout for debug purpose
	jsonBody, err := json.MarshalIndent(pub, " ", "  ")
	if err != nil {
		pub.ErrorMessage = "Error creating json pub"
		exitWithError(pub, err, 70)
	}
	os.Stdout.Write(jsonBody)
	os.Stdout.WriteString("\nEncryption was successful\n")
	os.Exit(0)
}
