// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package logging

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/readium/readium-lcp-server/config"
	"github.com/slack-go/slack"
)

var (
	LogFile        *log.Logger
	SlackApi       *slack.Client
	SlackChannelID string
)

// Init inits the log file and opens it
func Init(logging config.Logging) error {

	//logPath string, cm bool
	if logging.Directory != "" {
		log.Println("Open log file as " + logging.Directory)
		file, err := os.OpenFile(logging.Directory, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}

		LogFile = log.New(file, "", log.LUTC)
	}
	if logging.SlackToken != "" && logging.SlackChannelID != "" {
		log.Println("Init Slack connection")
		SlackApi = slack.New(logging.SlackToken)
		if SlackApi == nil {
			return errors.New("error creating a Slack connector")
		}
		SlackChannelID = logging.SlackChannelID
	}
	return nil
}

// Print writes a message to the log file / Slack
func Print(message string) {
	// log on stdout
	log.Print(message)

	// log on a file
	if LogFile != nil {
		currentTime := time.Now().UTC().Format(time.RFC3339)
		LogFile.Println(currentTime + "\t" + message)
	}

	// log on Slack
	if SlackApi != nil {
		_, _, err := SlackApi.PostMessage(SlackChannelID, slack.MsgOptionText(message, false))
		if err != nil {
			log.Printf("Error sending Slack msg, %v", err)
		}
	}
}
