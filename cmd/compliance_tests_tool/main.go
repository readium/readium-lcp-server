/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

//This tool is for testers to control and log the tests

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	req, err := http.NewRequest("GET", publicBaseUrl+"/compliancetest", nil)
	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("test_stage", testStage)
	q.Add("test_number", testNumber)
	q.Add("test_result", result)
	req.URL.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		log.Printf("Error Notify LsdServer of compliancetest: %v", err)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Notify LsdServer of compliancetest reading body error : %v", err)
	}

	log.Printf("Lsd Server on compliancetest response : %v [http-status:%d]", body, resp.StatusCode)
}
