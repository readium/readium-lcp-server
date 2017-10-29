// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

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
	BASIC_FUNCTION       = "2.3.2.1"
	SUCCESS_REGISTRATION = "2.3.2.2.1"
	REJECT_REGISTRATION  = "2.3.2.2.2"
	SUCCESS_RETURN       = "2.3.2.3.1"
	REJECT_RETURN        = "2.3.2.3.2"
	SUCCESS_RENEW        = "2.3.2.4.1"
	REJECT_RENEW         = "2.3.2.4.2"
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
