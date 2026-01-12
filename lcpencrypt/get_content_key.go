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
	EncryptionKey []byte `json:"key,omitempty"`
}

type PubInfo struct {
	EncryptionKey []byte    `json:"encryption_key"`
}


// getContentKey gets content information from the License Server
// for a given content id,
// and returns the associated content key.
func getContentKey(contentID, lcpsv string, v2 bool, username, password string) (string, error) {

	var contentKey string

	// An empty notify URL is not an error, simply a silent encryption
	if lcpsv == "" {
		return contentKey, nil
	}

	if !strings.HasPrefix(lcpsv, "http://") && !strings.HasPrefix(lcpsv, "https://") {
		lcpsv = "http://" + lcpsv
	}
	var getInfoURL string
	var err error
	if v2 {
		getInfoURL, err = url.JoinPath(lcpsv, "publications", contentID)
	} else {
		getInfoURL, err = url.JoinPath(lcpsv, "contents", contentID, "info")
	}
	if err != nil {
		return contentKey, nil
	}

	// look for the username and password in the url
	err = getUsernamePassword(&getInfoURL, &username, &password)
	if err != nil {
		return contentKey, nil
	}

	req, err := http.NewRequest("GET", getInfoURL, nil)
	if err != nil {
		return contentKey, nil
	}

	req.SetBasicAuth(username, password)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return contentKey, nil
	}
	defer resp.Body.Close()

	// if the content is found, the content key is updated
	if resp.StatusCode == http.StatusOK {

		if v2 {
			pubInfo := PubInfo{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&pubInfo)
			if err != nil {
				return contentKey, errors.New("unable to decode content information")
			}
			contentKey = b64.StdEncoding.EncodeToString(pubInfo.EncryptionKey)
		} else {
			contentInfo := ContentInfo{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&contentInfo)
			if err != nil {
				return contentKey, errors.New("unable to decode content information")
			}
			contentKey = b64.StdEncoding.EncodeToString(contentInfo.EncryptionKey)
		}
		fmt.Println("Existing encryption key retrieved")
	} else if resp.StatusCode != http.StatusNotFound {
		return contentKey, fmt.Errorf("the server returned an error %d", resp.StatusCode)
	}
	return contentKey, nil
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
