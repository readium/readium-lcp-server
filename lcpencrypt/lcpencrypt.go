// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/readium/readium-lcp-server/encrypt"
)

// showHelpAndExit displays some help and exits.
func showHelpAndExit() {

	fmt.Println("lcpencrypt protects a publication using the LCP DRM")
	fmt.Println("-input        source epub/pdf/lpf file locator (file system or http GET)")
	fmt.Println("[-storage]    optional, final storage of the encrypted publication, fs or s3")
	fmt.Println("[-url]        optional, base url associated with the storagen")
	fmt.Println("[-output]     optional, target path of encrypted publications")
	fmt.Println("[-contentid]  optional, content identifier; if omitted a uuid is generated")
	fmt.Println("[-lcpsv]      optional, http endpoint, notification of the License server")
	fmt.Println("[-login]      login (License server) ")
	fmt.Println("[-password]   password (License server)")
	fmt.Println("[-help] :     help information")
	os.Exit(0)
}

// exitWithError outputs an error message and exits.
func exitWithError(context string, err error) {

	fmt.Println(context, ":", err.Error())
	os.Exit(1)
}

func main() {
	var inputPath = flag.String("input", "", "source epub/pdf/lpf file locator (file system or http GET)")
	var storageRepo = flag.String("storage", "", "optional, final storage of the encrypted publication, fs or s3")
	var storageURL = flag.String("url", "", "optional, base url associated with the storage")
	var outputRepo = flag.String("output", "", "optional, target path of encrypted publications")
	var contentid = flag.String("contentid", "", "optional, content identifier; if omitted a uuid is generated")
	var lcpsv = flag.String("lcpsv", "", "optional, http endpoint, notification of the License server")
	var username = flag.String("login", "", "login (License server)")
	var password = flag.String("password", "", "password (License server)")

	var help = flag.Bool("help", false, "shows information")

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

	// encrypt the publication
	pub, err := encrypt.ProcessPublication(*contentid, *inputPath, *outputRepo, *storageRepo, *storageURL)
	if err != nil {
		exitWithError("Process a publication", err)
	}

	// notify the license server
	err = encrypt.NotifyLcpServer(pub, *lcpsv, *username, *password)
	if err != nil {
		exitWithError("Notify the LCP Server", err)
	}

	// write a json message to stdout for debug purpose
	jsonBody, err := json.MarshalIndent(pub, " ", "  ")
	if err != nil {
		exitWithError("Debug Message", errors.New("JSON error"))
	}
	fmt.Println("Encryption message:")
	os.Stdout.Write(jsonBody)
	fmt.Println("\nEncryption was successful)")
	os.Exit(0)
}
