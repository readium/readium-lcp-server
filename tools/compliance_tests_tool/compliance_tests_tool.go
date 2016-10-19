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
