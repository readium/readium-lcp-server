// Copyright 2021 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
)

// LCPServerMsgV2 is used for notifying an LCP Server V2
type LCPServerMsgV2 struct {
	UUID          string `json:"uuid"`
	Title         string `json:"title"`
	EncryptionKey []byte `json:"encryption_key"`
	Location      string `json:"location"`
	ContentType   string `json:"content_type"`
	Size          uint32 `json:"size"`
	Checksum      string `json:"checksum"`
}

type Coded struct {
	Code string `json:"code"`
}

type Entity struct {
	Name string `json:"name"`
}

// CMSMsg is used for notifying a CMS
type CMSMsg struct {
	UUID          string   `json:"uuid"`
	Title         string   `json:"title"`
	ContentType   string   `json:"content_type"`
	DatePublished string   `json:"date_published"`
	Description   string   `json:"description"`
	CoverUrl      string   `json:"cover_url,omitempty"`
	Language      []Coded  `json:"language"`
	Publisher     []Entity `json:"publisher"`
	Author        []Entity `json:"author"`
	Category      []Entity `json:"category"`
}

// NotifyLCPServer notifies the License Server of the encryption of a publication
func NotifyLCPServer(pub Publication, lcpsv string, v2 bool, username string, password string, verbose bool) error {

	// An empty notify URL is not an error, simply a silent encryption
	if lcpsv == "" {
		return nil
	}

	if !strings.HasPrefix(lcpsv, "http://") && !strings.HasPrefix(lcpsv, "https://") {
		lcpsv = "http://" + lcpsv
	}
	var notifyURL string
	var err error
	if v2 {
		notifyURL, err = url.JoinPath(lcpsv, "publications")
	} else {
		notifyURL, err = url.JoinPath(lcpsv, "contents", pub.UUID)
	}
	if err != nil {
		return err
	}

	// look for the username and password in the url
	err = getUsernamePassword(&notifyURL, &username, &password)
	if err != nil {
		return err
	}

	var req *http.Request
	var jsonBody []byte

	// the payload sent to the server differs from v1 to v2 servers
	if !v2 {

		var msg apilcp.Encrypted
		msg.ContentID = pub.UUID
		msg.ContentKey = pub.EncryptionKey
		msg.StorageMode = pub.StorageMode
		msg.Output = pub.Location
		msg.FileName = pub.FileName
		msg.ContentType = pub.ContentType
		msg.Size = int64(pub.Size)
		msg.Checksum = pub.Checksum

		jsonBody, err = json.Marshal(msg)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("PUT", notifyURL, bytes.NewReader(jsonBody))
		if err != nil {
			return err
		}
	} else {
		var msg LCPServerMsgV2
		msg.UUID = pub.UUID
		msg.Title = pub.Title
		msg.EncryptionKey = pub.EncryptionKey
		msg.Location = pub.Location
		msg.ContentType = pub.ContentType
		msg.Size = pub.Size
		msg.Checksum = pub.Checksum

		jsonBody, err = json.Marshal(msg)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("POST", notifyURL, bytes.NewReader(jsonBody))
		if err != nil {
			return err
		}
	}

	// verbose: log the notification
	if verbose {
		fmt.Println("LCP Server Notification:")
		var out bytes.Buffer
		json.Indent(&out, jsonBody, "", " ")
		out.WriteTo(os.Stdout)
		fmt.Println("")
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

	if (resp.StatusCode != 302) && (resp.StatusCode/100) != 2 { //302=found or 20x reply = OK
		return fmt.Errorf("the server returned an error %d", resp.StatusCode)
	}
	fmt.Println("The LCP Server was notified")
	return nil
}

// AbortNotification rolls back the notification of the encryption of a publication
func AbortNotification(pub Publication, lcpsv string, v2 bool, username string, password string) error {

	// An empty notify URL is not an error
	if lcpsv == "" {
		return nil
	}

	if !strings.HasPrefix(lcpsv, "http://") && !strings.HasPrefix(lcpsv, "https://") {
		lcpsv = "http://" + lcpsv
	}
	var notifyURL string
	var err error
	if v2 {
		notifyURL, err = url.JoinPath(lcpsv, "publications", pub.UUID)
	} else {
		notifyURL, err = url.JoinPath(lcpsv, "contents", pub.UUID)
	}
	if err != nil {
		return err
	}

	// look for the username and password in the url
	err = getUsernamePassword(&notifyURL, &username, &password)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", notifyURL, nil)
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

	if (resp.StatusCode != 302) && (resp.StatusCode/100) != 2 { //302=found or 20x reply = OK
		return fmt.Errorf("the server returned an error %d", resp.StatusCode)
	}
	fmt.Println("Encrypted publication deleted from the LCP Server")
	return nil
}

// NotifyCMS notifies a CMS of the encryption of a publication
func NotifyCMS(pub Publication, notifyURL string, verbose bool) error {

	// An empty notify URL is not an error, simply a silent encryption
	if notifyURL == "" {
		return nil
	}

	// look for the clientID and clientSecret in the url
	var username, password string
	err := getUsernamePassword(&notifyURL, &username, &password)
	if err != nil {
		return err
	}

	// set the message sent to the CMS
	var msg CMSMsg
	msg.UUID = pub.UUID
	msg.Title = pub.Title
	msg.ContentType = pub.ContentType
	msg.DatePublished = pub.Date
	msg.Description = pub.Description
	msg.CoverUrl = pub.CoverUrl
	var lg Coded
	for _, v := range pub.Language {
		lg.Code = v
		msg.Language = append(msg.Language, lg)
	}
	var en Entity
	for _, en.Name = range pub.Publisher {
		msg.Publisher = append(msg.Publisher, en)
	}
	for _, en.Name = range pub.Author {
		msg.Author = append(msg.Author, en)
	}
	for _, en.Name = range pub.Subject {
		msg.Category = append(msg.Category, en)
	}

	var req *http.Request
	var jsonBody []byte

	jsonBody, err = json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err = http.NewRequest("POST", notifyURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	// verbose: display the notification
	if verbose {
		fmt.Println("CMS Notification:")
		var out bytes.Buffer
		json.Indent(&out, jsonBody, "", " ")
		out.WriteTo(os.Stdout)
		fmt.Println("")
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

	if (resp.StatusCode != 302) && (resp.StatusCode/100) != 2 { //302=found or 20x reply = OK
		return fmt.Errorf("the server returned an error %d", resp.StatusCode)
	}
	fmt.Println("The CMS was notified")
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
