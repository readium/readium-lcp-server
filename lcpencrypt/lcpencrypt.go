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
	"github.com/readium/readium-lcp-server/lcpserver/api"
	"github.com/readium/readium-lcp-server/pack"
)

// notification of newly added content (Publication)
func notifyLcpServer(lcpService, contentid string, lcpPublication apilcp.LcpPublication) error {
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
	log.Println("lcpencrypt protects an epub file for usage in an lcp environment")
	log.Println("-input : source epub file locator (file system or http GET)")
	log.Println("[-contentid] : optional content identifier, if omitted a new one will be generated")
	log.Println("[-output] : optional target location for protected content (file system or http PUT)")
	log.Println("[-lcpsv] : optional http endpoint of the LCP server")
	log.Println("[-help] : help information")
	os.Exit(0)
	return
}

func exitWithError(lcpPublication apilcp.LcpPublication, err error, errorlevel int) {
	os.Stderr.WriteString(lcpPublication.ErrorMessage)
	os.Stderr.WriteString("\n")
	os.Stderr.WriteString(err.Error())
	jsonBody, err := json.MarshalIndent(lcpPublication, " ", "  ")
	if err != nil {
		os.Stderr.WriteString("\nError creating json lpcPublication")
		os.Exit(errorlevel)
	}
	os.Stdout.WriteString("\nlpcPublication:\n")
	os.Stdout.Write(jsonBody)
	os.Exit(errorlevel)
}

func main() {
	var err error
	var addedPublication apilcp.LcpPublication
	var inputFilename = flag.String("input", "", "source epub file locator (file system or http GET)")
	var contentid = flag.String("contentid", "", "optional content identifier; if omitted a new one is generated")
	var outputFilename = flag.String("output", "", "optional target location for protected content (file system or http PUT)")
	var lcpsv = flag.String("lcpsv", "", "optional http endpoint of the LCP server")
	var help = flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}
	if *help {
		showHelpAndExit()
	}

	// read the epub input file content in memory
	buf, err := getInputFile(*inputFilename)
	if err != nil {
		addedPublication.ErrorMessage = "Error opening input file, for more information type 'lcpencrypt -help' "
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
		*outputFilename = strings.Join([]string{workingDir, string(os.PathSeparator), *contentid, ".epub"}, "")
	}
	addedPublication.Output = *outputFilename

	// read the epub content from the zipped buffer
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		addedPublication.ErrorMessage = "Error opening zip/epub"
		exitWithError(addedPublication, err, 10)
		return
	}
	ep, err := epub.Read(zr)
	if err != nil {
		addedPublication.ErrorMessage = "Error reading epub content"
		exitWithError(addedPublication, err, 8)
		os.Exit(3)
		return
	}

	// create an output file
	output, err := os.Create(*outputFilename)
	if err != nil {
		addedPublication.ErrorMessage = "Error writing output file"
		exitWithError(addedPublication, err, 4)
		return
	}

	// pack / encrypt the epub content, fill the output file
	_, encryptionKey, err := pack.Do(ep, output)
	output.Close()
	if err != nil {
		addedPublication.ErrorMessage = "Error packing"
		exitWithError(addedPublication, err, 6)
		return
	}
	addedPublication.ContentKey = encryptionKey
	addedPublication.Output = *outputFilename

	// notify the LCP Server
	if *lcpsv != "" {
		err = notifyLcpServer(*lcpsv, *contentid, addedPublication)
		if err != nil {
			addedPublication.ErrorMessage = "Error updating LCP-server"
			exitWithError(addedPublication, err, 1)
			os.Exit(6)
		}
	}

	// write json message to stdout
	jsonBody, err := json.Marshal(addedPublication)
	if err != nil {
		addedPublication.ErrorMessage = "Error creating json addedPublication"
		exitWithError(addedPublication, err, 1)
		return
	}
	os.Stdout.Write(jsonBody)
	os.Stdout.WriteString("\n")

	os.Exit(0)
}
