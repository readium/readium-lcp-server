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

	fmt.Println("lcpencrypt encrypts a publication using the LCP DRM")
	fmt.Println("-input        source epub/pdf/lpf file locator (file system or http GET)")
	fmt.Println("[-contentid]  optional, content identifier; if omitted a uuid is generated")
	fmt.Println("[-storage]    optional, target location of the encrypted publication, without filename. File system path or s3 bucket")
	fmt.Println("[-url]        optional, base url associated with the storage, without filename")
	fmt.Println("[-filename]   optional, file name of the encrypted publication; if omitted, contentid is used")
	fmt.Println("[-output]     optional, target folder of encrypted publications")
	fmt.Println("[-temp]       optional, working folder for temporary files")
	fmt.Println("[-contentkey]  optional, base64 encoded content key; if omitted a random content key is generated")
	fmt.Println("[-lcpsv]      optional, http endpoint, notification of the License server")
	fmt.Println("[-login]      login (License server) ")
	fmt.Println("[-password]   password (License server)")
	fmt.Println("[-v2sv]       optional, indicates that the license server is in v2, values 0 (default) or 1")
	fmt.Println("[-help] :     help information")
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
	contentkey := flag.String("contentkey", "", "optional, base64 encoded content key; if omitted a random content key is generated")
	lcpsv := flag.String("lcpsv", "", "optional, http endpoint, notification of the License server")
	username := flag.String("login", "", "login (License server)")
	password := flag.String("password", "", "password (License server)")
	v2sv := flag.Bool("v2sv", false, "optional, compatibility with the License Server v2")

	help := flag.Bool("help", false, "shows information")

	if !flag.Parsed() {
		flag.Parse()
	}

	if *help || *inputPath == "" {
		showHelpAndExit()
	}

	if *lcpsv != "" && (*username == "" || *password == "") {
		exitWithError("Parameters", errors.New("incorrect parameters, lcpsv needs a login and password, for more information type 'lcpencrypt -help' "))
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

	// encrypt the publication
	pub, err := encrypt.ProcessEncryption(*contentid, *contentkey, *inputPath, *tempRepo, *outputRepo, *storageRepo, *storageURL, *storageFilename)
	if err != nil {
		exitWithError("Error processing a publication", err)
	}

	elapsed := time.Since(start)
	fmt.Println("Encryption took ", elapsed)

	// notify the license server
	err = encrypt.NotifyLcpServer(pub, *lcpsv, *username, *password, *v2sv)
	if err != nil {
		exitWithError("Notify the LCP Server", err)
	}

	fmt.Println("\nEncryption was successful.")
	os.Exit(0)
}
