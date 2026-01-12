// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/readium/readium-lcp-server/encrypt"
)

const (
	// DO NOT FORGET to update the version
	Software_Version = "1.12.7"
)

// showHelpAndExit displays some help and exits.
func showHelpAndExit() {

	fmt.Println("lcpencrypt encrypts a publication using the LCP DRM.")
	fmt.Println("-input      source epub/pdf/lpf/audiobook file locator (file system or http GET)")
	fmt.Println("-provider   publication provider (URI)")
	fmt.Println("-storage    optional, target location of the encrypted publication, without filename. File system path or s3 bucket")
	fmt.Println("-url        optional, base url associated with the storage, without filename")
	fmt.Println("-filename   optional, file name for the encrypted publication; if omitted, contentid is used")
	fmt.Println("-temp       optional, working folder for temporary files. If not set, the current directory will be used.")
	fmt.Println("-cover      optional, boolean, indicates that a cover should be generated")
	fmt.Println("-pdfnometa  optional, boolean, indicates that PDF metadata must not be extracted")
	fmt.Println("-contentid  optional, publication identifier; if omitted a uuid is generated")
	fmt.Println("-lcpsv      optional, URL, host name of the License Server to be notified; syntax http://username:password@example.com")
	fmt.Println("-v2         optional, boolean, indicates communication with a License Server v2")
	// these parameters are deprecated, let's be silent about them in the help
	//fmt.Println("-login      optional, used along with lcpsv, username for the License server")
	//fmt.Println("-password   optional, used along with lcpsv, password for the License server")
	fmt.Println("-notify     optional, URL, notification endpoint of a CMS; syntax http://username:password@example.com")
	fmt.Println("-verbose    optional, boolean, the information sent to the LCP Server and CMS will be displayed")
	fmt.Println("-output     optional, deprecated, temporary location of encrypted publications, before the License Server moves them. File system path only. This path must be directly accessible from the License Server. If not set, encrypted publications will be temporarily created into the current directory.")
	fmt.Println("-help :     help information")
	os.Exit(0)
}

// exitWithError outputs an error message and exits.
func exitWithError(context string, err error) {

	fmt.Println(context, ":", err.Error())
	os.Exit(1)
}

func main() {
	inputPath := flag.String("input", "", "source epub/pdf/lpf file locator (file system or http GET)")
	providerUri := flag.String("provider", "", "publication provider (URI)")
	storageRepo := flag.String("storage", "", "target location of the encrypted publication, without filename. File system path or s3 bucket")
	storageURL := flag.String("url", "", "base url associated with the storage, without filename")
	storageFilename := flag.String("filename", "", "file name of the encrypted publication; if omitted, contentid is used")
	outputRepo := flag.String("output", "", "target folder of encrypted publications")
	tempRepo := flag.String("temp", "", "working folder for temporary files")
	cover := flag.Bool("cover", false, "boolean, indicates that covers must be generated when possible")
	pdfnometa := flag.Bool("pdfnometa", false, "boolean, indicates that PDF metadata must not be extracted")
	useFilenameAs := flag.String("usefnas", "", "if set to 'uuid', the file name is used as publication uuid")
	contentid := flag.String("contentid", "", "imposed publication UUID, used to update an existing publication")
	lcpsv := flag.String("lcpsv", "", "URL, host name of the License server which is notified; the preferred syntax is http://username:password@example.com")
	v2 := flag.Bool("v2", false, "boolean, indicates a v2 License server")
	username := flag.String("login", "", "optional unless lcpsv is used, username for the License server")
	password := flag.String("password", "", "optional unless lcpsv is used, password for the License server")
	notify := flag.String("notify", "", "URL, notification endpoint for a CMS; its syntax is http://username:password@example.com")
	verbose := flag.Bool("verbose", false, "boolean, the information sent to the LCP Server and CMS will be displayed")

	help := flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}

	if *help || *inputPath == "" {
		showHelpAndExit()
	}

	if *storageRepo != "" && *storageURL == "" {
		exitWithError("Parameters", errors.New("incorrect parameters, storage requires url, for more information type 'lcpencrypt -help' "))
	}
	if filepath.Ext(*outputRepo) != "" {
		exitWithError("Parameters", errors.New("incorrect parameters, output must not contain a file name, for more information type 'lcpencrypt -help' "))
	}
	if filepath.Ext(*storageRepo) != "" {
		exitWithError("Parameters", errors.New("incorrect parameters, storage must not contain a file name, for more information type 'lcpencrypt -help' "))
	}

	start := time.Now()

	// get the file name from the input path, strip the extension
	filename := strings.TrimSuffix(filepath.Base(*inputPath), filepath.Ext(*inputPath))

	// if the publication UUID is imposed, check if the content already exists in the License Server.
	// Note that the publication UUID may also have be set via the command line. 
	// If this is the case, get the content encryption key for the server, so that the new encryption
	// keeps the same key.
	// This is necessary to allow fresh licenses being capable of decrypting previously downloaded content.
	if *useFilenameAs == "uuid" {
		*contentid = filename
	}

	var contentkey string
	if *contentid != "" {
		// warning: this is a synchronous REST call
		// contentKey is not initialized if the content does not exist in the License Server
		var err error
		contentkey, err = getContentKey(*contentid, *lcpsv, *v2, *username, *password)
		if err != nil {
			exitWithError("Error retrieving content info", err)
		}
	}

	// encrypt the publication
	publication, err := encrypt.ProcessEncryption(*contentid, contentkey, *inputPath, *tempRepo, *outputRepo, *storageRepo, *storageURL, *storageFilename, *cover, *pdfnometa)
	if err != nil {
		exitWithError("Error processing a publication", err)
	}

	// logs if verbose mode
	if *verbose {
		log.Println("Software Version " + Software_Version)
		log.Println("Encrypted file:", filepath.Join(publication.OutputRepo, publication.FileName))
		if publication.ExtractCover {
			log.Println("Cover file name:", publication.CoverName)
			log.Println("Cover file url:", publication.CoverUrl)
		}
	}

	elapsed := time.Since(start)

	// notify the license server
	err = encrypt.NotifyLCPServer(*publication, contentkey != "", *providerUri, *lcpsv, *v2, *username, *password, *verbose)
	if err != nil {
		exitWithError("Error notifying the LCP Server", err)
	}

	// notify a CMS (username and password are always in the URL)
	err = encrypt.NotifyCMS(*publication, *notify, *verbose)
	if err != nil {
		fmt.Println("Error notifying the CMS:", err.Error())
		// abort the notification of the license server
		err = encrypt.AbortNotification(*publication, *lcpsv, *v2, *username, *password)
		if err != nil {
			exitWithError("Error aborting notification of the LCP Server", err)
		}
	}

	fmt.Println("The encryption took", elapsed)
	os.Exit(0)
}
