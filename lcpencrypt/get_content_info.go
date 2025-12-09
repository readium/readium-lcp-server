// Copyright 2024 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ContentInfo struct {
	ID            string `json:"id"`
	EncryptionKey []byte `json:"key,omitempty"`
	Location      string `json:"location"`
	Length        int64  `json:"length"`
	Sha256        string `json:"sha256"`
	Type          string `json:"type"`
}

// getContentKey gets content information from the License Server
// for a given content id,
// and returns the associated content key.
func getContentKey(contentKey *string, contentID, lcpsv string, v2 bool, username, password string) error {

	// An empty notify URL is not an error, simply a silent encryption
	if lcpsv == "" {
		return nil
	}

	if !strings.HasPrefix(lcpsv, "http://") && !strings.HasPrefix(lcpsv, "https://") {
		lcpsv = "http://" + lcpsv
	}
	var getInfoURL string
	var err error
	if v2 {
		getInfoURL, err = url.JoinPath(lcpsv, "publications", contentID, "info")
	} else {
		getInfoURL, err = url.JoinPath(lcpsv, "contents", contentID, "info")
	}
	if err != nil {
		return err
	}

	// look for the username and password in the url
	err = getUsernamePassword(&getInfoURL, &username, &password)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", getInfoURL, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(username, password)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// if the content is found, the content key is updated
	if resp.StatusCode == http.StatusOK {
		contentInfo := ContentInfo{}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&contentInfo)
		if err != nil {
			return errors.New("unable to decode content information")
		}

		*contentKey = b64.StdEncoding.EncodeToString(contentInfo.EncryptionKey)
		fmt.Println("Existing encryption key retrieved")
	}
	return nil
}

// Look for the username and password in the url
func getUsernamePassword(notifyURL, username, password *string) error {
	u, err := url.Parse(*notifyURL)
	if err != nil {
		return err
	}
	un := u.User.Username()
	pw, pwfound := u.User.Password()
	if un != "" && pwfound {
		*username = un
		*password = pw
		u.User = nil
		*notifyURL = u.String() // notifyURL is updated
	}
	return nil
}
