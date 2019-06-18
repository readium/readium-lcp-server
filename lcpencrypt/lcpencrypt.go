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
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/lcpserver/api"
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

// reads and returns the content of
// a file on the local filesystem
// or via a GET if the scheme is http:// or https://
func getInputFile(inputFilename string) ([]byte, error) {
	url, err := url.Parse(inputFilename)
	if err != nil {
		return nil, errors.New("Error parsing input file")
	}
	if url.Scheme == "http" || url.Scheme == "https" {
		res, err := http.Get(inputFilename)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		return ioutil.ReadAll(res.Body)
	} else if url.Scheme == "ftp" {
		return nil, errors.New("ftp not supported yet")

	} else {
		return ioutil.ReadFile(inputFilename)
	}
}

func showHelpAndExit() {
	log.Println("lcpencrypt protects an epub/pdf file for usage in an lcp environment")
	log.Println("-input        source epub/pdf file locator (file system or http GET)")
	log.Println("[-profile]    encryption profile to use")
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
	os.Stdout.WriteString(lcpPublication.ErrorMessage)
	os.Stdout.WriteString("\n")
	if err != nil {
		os.Stdout.WriteString(err.Error())
	}
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

func getChecksum(filename string) string {
	hasher := sha256.New()
	s, err := ioutil.ReadFile(filename)
	hasher.Write(s)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func OutputExtension(sourceExt string) string {
	if sourceExt == ".pdf" {
		return ".lcpdf"
	} else {
		return ".epub"
	}
}

func main() {
	var err error
	var addedPublication apilcp.LcpPublication
	var inputFilename = flag.String("input", "", "source epub/pdf file locator (file system or http GET)")
	var contentid = flag.String("contentid", "", "optional content identifier; if omitted a new one is generated")
	var outputFilename = flag.String("output", "", "optional target location for the encrypted content (file system or http PUT)")
	var lcpsv = flag.String("lcpsv", "", "optional http endpoint of the License server (adds content)")
	var username = flag.String("login", "", "login (License server)")
	var password = flag.String("password", "", "password (License server)")
	var profile = flag.String("profile", "basic", "LCP Profile to use for encryption")

	var help = flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}
	if *help {
		showHelpAndExit()
	}

	if *lcpsv != "" && (*username == "" || *password == "") {
		addedPublication.ErrorMessage = "incorrect parameters, lcpsv needs login and password, for more information type 'lcpencrypt -help' "
		exitWithError(addedPublication, nil, 80)
	}

	if *contentid == "" { // contentID not set -> generate a new one
		uid, err_u := uuid.NewV4()
		if err_u != nil {
			exitWithError(addedPublication, err, 65)
		}
		*contentid = uid.String()
	}
	var basefilename string
	addedPublication.ContentId = *contentid
	// if the output file name not set,
	// then <content-id>.epub|lcpdf is created in the working directory
	if *outputFilename == "" {
		workingDir, _ := os.Getwd()
		ext := filepath.Ext(*inputFilename)
		outputExt := OutputExtension(ext)
		*outputFilename = strings.Join([]string{workingDir, string(os.PathSeparator), *contentid, outputExt}, "")
		basefilename = filepath.Base(*inputFilename)
	} else {
		basefilename = filepath.Base(*outputFilename)
	}
	addedPublication.ContentDisposition = &basefilename
	// the output path must be accessible from the license server
	addedPublication.Output = *outputFilename

	var lcpProfile pack.EncryptionProfile
	if *profile == "v1" {
		lcpProfile = pack.EncryptionProfile(license.V1_PROFILE)
	} else {
		lcpProfile = pack.EncryptionProfile(license.BASIC_PROFILE)
	}

	var output *os.File
	var encryptionKey crypto.ContentKey
	encrypter := crypto.NewAESEncrypter_PUBLICATION_RESOURCES()
	if strings.HasSuffix(*inputFilename, ".epub") {
		addedPublication.ContentType = epub.ContentType_EPUB
		// read the input file content in memory
		buf, err := getInputFile(*inputFilename)
		if err != nil {
			addedPublication.ErrorMessage = "Error opening input file, for more information type 'lcpencrypt -help' "
			exitWithError(addedPublication, err, 70)
		}
		// read the epub content from the zipped buffer
		zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
		if err != nil {
			addedPublication.ErrorMessage = "Error opening the epub file"
			exitWithError(addedPublication, err, 60)
		}
		ep, err := epub.Read(zr)
		if err != nil {
			addedPublication.ErrorMessage = "Error reading the epub content"
			exitWithError(addedPublication, err, 50)
		}

		// create an output file
		output, err = os.Create(*outputFilename)
		if err != nil {
			addedPublication.ErrorMessage = "Error writing output file"
			exitWithError(addedPublication, err, 40)
		}

		// pack / encrypt the epub content, fill the output file
		_, encryptionKey, err = pack.Do(encrypter, ep, output)
	} else if strings.HasSuffix(*inputFilename, ".pdf") {
		addedPublication.ContentType = "application/pdf+lcp"
		packagePath := *outputFilename + ".webpub"
		err := pack.BuildWebPubPackageFromPDF(filepath.Base(*inputFilename), *inputFilename, packagePath)
		if err != nil {
			addedPublication.ErrorMessage = "Error building Web Publication package from PDF"
			exitWithError(addedPublication, err, 50)
		}

		// create an output file
		output, err = os.Create(*outputFilename)
		if err != nil {
			addedPublication.ErrorMessage = "Error writing output file"
			exitWithError(addedPublication, err, 40)
		}

		reader, err := pack.OpenPackagedRWP(packagePath)
		if err != nil {
			addedPublication.ErrorMessage = "Error opening packaged web publication"
			exitWithError(addedPublication, err, 40)
		}

		writer, err := reader.NewWriter(output)
		if err != nil {
			addedPublication.ErrorMessage = "Error opening output"
			exitWithError(addedPublication, err, 40)
		}

		encryptionKey, err = pack.Process(lcpProfile, encrypter, reader, writer)
		if err != nil {
			addedPublication.ErrorMessage = "Error encrypting"
			exitWithError(addedPublication, err, 40)
		}

		// Remove the temporary package file
		os.Remove(packagePath)
	}

	stats, err := output.Stat()
	if err == nil && (stats.Size() > 0) {
		filesize := stats.Size()
		cs := getChecksum(*outputFilename)
		addedPublication.Size = &filesize
		addedPublication.Checksum = &cs
	}
	output.Close()
	if err != nil || (stats.Size() == 0) {
		addedPublication.ErrorMessage = "Error encrypting the publication"
		exitWithError(addedPublication, err, 30)
	}
	addedPublication.ContentKey = encryptionKey

	// notify the LCP Server
	if *lcpsv != "" {
		err = notifyLcpServer(*lcpsv, *contentid, addedPublication, *username, *password)
		if err != nil {
			addedPublication.ErrorMessage = "Error notifying the License Server"
			exitWithError(addedPublication, err, 20)
		} else {
			os.Stdout.WriteString("License Server was notified\n")
		}
	}

	// write a json message to stdout for debug purpose
	jsonBody, err := json.MarshalIndent(addedPublication, " ", "  ")
	if err != nil {
		addedPublication.ErrorMessage = "Error creating json addedPublication"
		exitWithError(addedPublication, err, 10)
	}
	os.Stdout.Write(jsonBody)
	os.Stdout.WriteString("\nEncryption was successful\n")
	os.Exit(0)
}
