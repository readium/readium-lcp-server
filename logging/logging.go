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

package logging

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	LogFile        *log.Logger
	complianceMode bool
)

const (
	BASIC_FUNCTION       = "2.3.2.1"
	SUCCESS_REGISTRATION = "2.3.2.2.1"
	REJECT_REGISTRATION  = "2.3.2.2.2"
	SUCCESS_RETURN       = "2.3.2.3.1"
	REJECT_RETURN        = "2.3.2.3.2"
	SUCCESS_RENEW        = "2.3.2.4.1"
	REJECT_RENEW         = "2.3.2.4.2"
)

//Init inits log file and opens it
func Init(logPath string, cm bool) error {
	complianceMode = cm
	if complianceMode == true {
		file, err := os.OpenFile(logPath, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		LogFile = log.New(file, "", log.LUTC)

		return nil
	}

	return nil
}

//WriteToFile writes result of function execution in log file
func WriteToFile(result string, status int, testId string) {
	if complianceMode == true {
		currentTime := time.Now().UTC().Format(time.RFC3339)

		var parsedStatus string
		if status != 0 {
			parsedStatus = strconv.Itoa(status)
		}
		LogFile.Println(result + "   " + currentTime + "   " + parsedStatus + "   " + testId)
	}
}

//ReadLogs reads logs from file
func ReadLogs(logPath string) ([]string, error) {
	var lines []string
	file, err := os.OpenFile(logPath, os.O_RDONLY, 0666)
	if err == nil {
		reader := bufio.NewReader(file)
		contents, err := ioutil.ReadAll(reader)

		if err == nil {
			lines = strings.Split(string(contents), "\n")
		}
	}
	lines = lines[:len(lines)-1]
	return lines, err
}

//CountTotal summarize the data in log file
func CountTotal(lines []string) (string, error) {
	var total, success, negative int
	var result string

	for _, value := range lines {
		splitted := strings.Split(value, "   ")

		if splitted[0] == "error" {
			negative++
		} else {
			success++
		}

		total++
	}

	result = "Total count: " + strconv.Itoa(total) + "; ended successfully: " + strconv.Itoa(success) + "; ended with error: " + strconv.Itoa(negative)
	return result, nil
}
