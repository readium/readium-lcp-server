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

//This tool is for testers to control and log the tests
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	lsdPublicBaseUrl := flag.String("lsdPublicBaseUrl", "", "public base url of lsd server")
	flag.Parse()

	var testNumber string
	var result string

	for {
		fmt.Println("Enter the number of test, 'q' for quit the tool")
		fmt.Scanln(&testNumber)

		if strings.EqualFold(testNumber, "q") {
			os.Exit(0)
		}

		notifyLsdServer(testNumber, "", "start", *lsdPublicBaseUrl)

		for {
			fmt.Println("Enter the result of test ('e' if test has errors, 's' if test has success)")
			fmt.Scanln(&result)

			if strings.EqualFold(result, "e") || strings.EqualFold(result, "s") {
				notifyLsdServer(testNumber, result, "end", *lsdPublicBaseUrl)
				break
			}
		}
	}
}

func notifyLsdServer(testNumber string, result string, testStage string, publicBaseUrl string) {
	var lsdClient = &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", publicBaseUrl+"/compliancetest", nil)
	q := req.URL.Query()
	q.Add("test_stage", testStage)
	q.Add("test_number", testNumber)
	q.Add("test_result", result)
	req.URL.RawQuery = q.Encode()

	response, err := lsdClient.Do(req)

	if err != nil {
		log.Println("Error Notify LsdServer of compliancetest: " + err.Error())
	} else {
		if response.StatusCode != 200 {
			log.Println("Notify LsdServer of compliancetest  = " + strconv.Itoa(response.StatusCode))
		}
	}
}
