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

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/readium/readium-lcp-server/epub"
	"github.com/readium/readium-lcp-server/pack"
)

// struct for communication with lcp-server
type LcpPublication struct {
	ContentId    string `json:"content-id"`
	ContentKey   []byte `json:"content-encryption-key"`
	Output       string `json:"protected-content-location"`
	ErrorMessage string `json:"error"`
}

// notification of newly added content (Publication)
func notifyLcpServer(lcpService, contentid string, lcpPublication LcpPublication) error {
	//exchange encryption key with lcp service/content/<id>,
	//Payload: {content-encryption-key, protected-content-location}
	//fmt.Printf("lcpsv = %s\n", *lcpsv)
	var urlBuffer bytes.Buffer
	urlBuffer.WriteString(lcpService)
	urlBuffer.WriteString("/content/")
	urlBuffer.WriteString(contentid)

	jsonBody, err := json.Marshal(lcpPublication)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", urlBuffer.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if (resp.StatusCode / 100) != 2 {
		return errors.New(fmt.Sprintf("lcp server error %d ", resp.StatusCode))
	}

	return nil
}

// reads and returns the content of
// a file on the local filesystem
// or via a GET if the scheme is http://   or https://
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
	log.Println("lcpencrypt packs en epub for usage in an lcp environment")
	log.Println("-input : source file locator.  (file system or http GET)")
	log.Println("[-contentid] : optional content identifier, if not present a new one is generated")
	log.Println("[-output] : optional target file for protected content (file system or http PUT)")
	log.Println("[-lcpsv] : http endpoint of the LCP service for exchange of information")
	log.Println("[-help] ")
	os.Exit(0)
	return
}

func exitWithError(lcpPublication LcpPublication, err error, errorlevel int) {
	os.Stderr.WriteString(lcpPublication.ErrorMessage)
	os.Stderr.WriteString(err.Error())
	os.Stderr.WriteString("\n")
	jsonBody, err := json.MarshalIndent(lcpPublication, " ", "  ")
	if err != nil {
		os.Stderr.WriteString("Error writing json to stdout")
		os.Exit(errorlevel)
	}
	os.Stdout.Write(jsonBody)
	os.Exit(errorlevel)
}

func main() {
	var err error
	var addedPublication LcpPublication
	var inputFilename = flag.String("input", "", "source file locator.  (file system or http GET)")
	var contentid = flag.String("contentid", "", "optional content identifier, if not present a new one is generated")
	var outputFilename = flag.String("output", "", "optional target file for protected content (file system or http PUT) ")
	var lcpsv = flag.String("lcpsv", "", "http endpoint of the LCP service for exchange of information ")
	var help = flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}
	if *help {
		showHelpAndExit()
	}
	buf, err := getInputFile(*inputFilename)
	if err != nil {
		addedPublication.ErrorMessage = "Error opening input, for more information type \"lcpencrypt -help\""
		exitWithError(addedPublication, err, 12)
		return
	}
	if *contentid == "" { // contentID not set -> generate a new one
		sha := sha256.Sum256(buf)
		*contentid = fmt.Sprintf("%x", sha)
	}
	addedPublication.ContentId = *contentid
	if *outputFilename == "" { //output not set -> "content-id.epub" in working directory
		workingDir, _ := os.Getwd()
		*outputFilename = strings.Join([]string{workingDir, *contentid, ".epub"}, "")
	}
	addedPublication.Output = *outputFilename
	// decode and pack epub file
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		addedPublication.ErrorMessage = "Error opening zip/epub"
		exitWithError(addedPublication, err, 10)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		addedPublication.ErrorMessage = "Error reading epub"
		exitWithError(addedPublication, err, 8)
		os.Exit(3)
		return
	}

	output, err := os.Create(*outputFilename)
	if err != nil {
		addedPublication.ErrorMessage = "Error writing output file"
		exitWithError(addedPublication, err, 4)
		return
	}

	_, encryptionKey, err := pack.Do(ep, output)
	output.Close()
	if err != nil {
		addedPublication.ErrorMessage = "Error packing"
		exitWithError(addedPublication, err, 6)
		return
	}
	addedPublication.ContentKey = encryptionKey
	addedPublication.Output = *outputFilename

	if *lcpsv != "" {
		err = notifyLcpServer(*lcpsv, *contentid, addedPublication)
		if err != nil {
			addedPublication.ErrorMessage = "Error updating LCP-server"
			exitWithError(addedPublication, err, 1)
			os.Exit(6)
		}
	}

	jsonBody, err := json.Marshal(addedPublication)
	if err != nil {
		addedPublication.ErrorMessage = "Error writing json to stdout"
		exitWithError(addedPublication, err, 1)
		return
	}
	os.Stdout.Write(jsonBody)
	os.Exit(0)
}
