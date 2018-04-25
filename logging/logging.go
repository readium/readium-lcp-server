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

// LogFile is the Logger file
var (
	LogFile        *log.Logger
	complianceMode bool
)

const (
	BasicFunction       = "2.3.2.1"
	SuccessRegistration = "2.3.2.2.1"
	RejectRegistration  = "2.3.2.2.2"
	SuccessReturn       = "2.3.2.3.1"
	RejectReturn        = "2.3.2.3.2"
	SuccessRenew        = "2.3.2.4.1"
	RejectRenew         = "2.3.2.4.2"
)

//Init inits the log file and opens it
func Init(logPath string, cm bool) error {
	complianceMode = cm
	if complianceMode == true {
		log.Println("Open compliance mode log file in " + logPath)
		file, err := os.OpenFile(logPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}

		LogFile = log.New(file, "", log.LUTC)

		return nil
	}

	return nil
}

//WriteToFile writes the result of the execution of a function to the log file
func WriteToFile(testID string, status string, result string, remark string) {
	if complianceMode == true {
		currentTime := time.Now().UTC().Format(time.RFC3339)

		LogFile.Println(currentTime + "\t" + testID + " \t" + status + "\t" + result + "\t" + remark)
	}
}

//ReadLogs reads all logs from the log file
func ReadLogs(logPath string) ([]string, error) {
	var lines []string
	file, err := os.OpenFile(logPath, os.O_RDONLY, 0666)
	if err == nil {
		reader := bufio.NewReader(file)
		// read all logs at once
		contents, err := ioutil.ReadAll(reader)
		if err == nil {
			// create the lines array
			lines = strings.Split(string(contents), "\n")
		}
	}
	// remove the last line (\n)
	lines = lines[:len(lines)-1]
	// close the file
	file.Close()
	return lines, err
}

//CountTotal summarizes data found in the log file
// TODO: compute useful info.
func CountTotal(lines []string) (string, error) {
	var total, positive, negative int
	var result string

	for _, value := range lines {
		splitted := strings.Split(value, "\t")

		if splitted[3] == "error" {
			negative++
		}
		if splitted[3] == "success" {
			positive++
		}

		total++
	}

	result = "Total count: " + strconv.Itoa(total) + "; ended successfully: " + strconv.Itoa(positive) + "; ended with error: " + strconv.Itoa(negative)
	return result, nil
}
