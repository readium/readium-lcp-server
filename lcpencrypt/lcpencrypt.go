// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/readium/readium-lcp-server/encrypt"
)

// showHelpAndExit displays some help and exits.
func showHelpAndExit() {

	fmt.Println("lcpencrypt encrypts a publication using the LCP DRM.")
	fmt.Println("-input      source epub/pdf/lpf/audiobook file locator (file system or http GET)")
	fmt.Println("-contentid  optional, content identifier; if omitted a uuid is generated")
	fmt.Println("-storage    optional, target location of the encrypted publication, without filename. File system path or s3 bucket")
	fmt.Println("-url        optional, base url associated with the storage, without filename")
	fmt.Println("-filename   optional, file name of the encrypted publication; if omitted, contentid is used")
	fmt.Println("-temp       optional, working folder for temporary files. If not set, the current directory will be used.")
	fmt.Println("-cover      optional, boolean, indicates that covers must be generated when possible")
	fmt.Println("-contentkey optional, base64 encoded content key; if omitted a random content key is generated")
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
	contentid := flag.String("contentid", "", "optional, content identifier; if omitted, a uuid is generated")
	storageRepo := flag.String("storage", "", "optional, target location of the encrypted publication, without filename. File system path or s3 bucket")
	storageURL := flag.String("url", "", "optional, base url associated with the storage, without filename")
	storageFilename := flag.String("filename", "", "optional, file name of the encrypted publication; if omitted, contentid is used")
	outputRepo := flag.String("output", "", "optional, target folder of encrypted publications")
	tempRepo := flag.String("temp", "", "optional, working folder for temporary files")
	cover := flag.Bool("cover", false, "optional, boolean, indicates that covers must be generated when possible")
	contentkey := flag.String("contentkey", "", "optional, base64 encoded content key; if omitted a random content key is generated")
	lcpsv := flag.String("lcpsv", "", "URL, host name of the License server which is notified; the preferred syntax is http://username:password@example.com")
	v2 := flag.Bool("v2", false, "optional, boolean, indicates a v2 License serve")
	username := flag.String("login", "", "optional unless lcpsv is used, username for the License server")
	password := flag.String("password", "", "optional unless lcpsv is used, password for the License server")
	notify := flag.String("notify", "", "optional, URL, notification endpoint for a CMS; its syntax is http://username:password@example.com")
	verbose := flag.Bool("verbose", false, "optional, boolean, the information sent to the LCP Server and CMS will be displayed")

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

	// if contentid is set but not contentkey, check if the content already exists in the License Server.
	// If this is the case, get the content encryption key for the server, so that the new encryption
	// keeps the same key. This is necessary to allow fresh licenses being capable
	// of decrypting previously downloaded content.
	if *contentid != "" && *contentkey == "" {
		// warning: this is a synchronous REST call
		// contentKey is not initialized if the content does not exist in the License Server
		err := getContentKey(contentkey, *contentid, *lcpsv, *v2, *username, *password)
		if err != nil {
			exitWithError("Error retrieving content info", err)
		}
	}

	// encrypt the publication
	publication, err := encrypt.ProcessEncryption(*contentid, *contentkey, *inputPath, *tempRepo, *outputRepo, *storageRepo, *storageURL, *storageFilename, *cover)
	if err != nil {
		exitWithError("Error processing a publication", err)
	}

	elapsed := time.Since(start)

	// notify the license server
	err = encrypt.NotifyLCPServer(*publication, *lcpsv, *v2, *username, *password, *verbose)
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
