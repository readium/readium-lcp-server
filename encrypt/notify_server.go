// Copyright 2021 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package encrypt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apilcp "github.com/readium/readium-lcp-server/lcpserver/api"
)

type PublicationV2 struct {
	UUID          string `json:"uuid"`
	Title         string `json:"title"`
	EncryptionKey []byte `json:"encryption_key"`
	Location      string `json:"location"`
	ContentType   string `json:"content_type"`
	Size          uint32 `json:"size"`
	Checksum      string `json:"checksum"`
}

// NotifyLcpServer notifies the License Server of the encryption of newly added publication
func NotifyLcpServer(pub *apilcp.LcpPublication, licenseServerURL string, username string, password string, v2sv bool) error {

	// An empty license server URL is not an error, simply a silent encryption
	if licenseServerURL == "" {
		fmt.Println("No notification sent to the License Server")
		return nil
	}

	var req *http.Request

	// prepare the call to the endpoint /contents/<id>
	var jsonBody []byte
	var err error
	if !v2sv {
		var urlBuffer bytes.Buffer
		urlBuffer.WriteString(licenseServerURL)
		urlBuffer.WriteString("/contents/")
		urlBuffer.WriteString(pub.ContentID)

		jsonBody, err = json.Marshal(*pub)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("PUT", urlBuffer.String(), bytes.NewReader(jsonBody))
		if err != nil {
			return err
		}

		// prepare the call to the endpoint /publications/<id>
	} else {
		url := licenseServerURL + "/publications/"

		var publication PublicationV2
		publication.UUID = pub.ContentID
		publication.Title = pub.FileName // the filename is sent, replaces a good title
		publication.EncryptionKey = pub.ContentKey
		publication.Location = pub.Output // url
		publication.ContentType = pub.ContentType
		publication.Size = uint32(pub.Size)
		publication.Checksum = pub.Checksum

		jsonBody, err = json.Marshal(publication)
		if err != nil {
			return err
		}
		req, err = http.NewRequest("POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return err
		}
	}

	/*
		// write a json message to stdout for debug purpose
		fmt.Println("Notification:")
		var out bytes.Buffer
		json.Indent(&out, jsonBody, "", " ")
		out.WriteTo(os.Stdout)
	*/

	req.SetBasicAuth(username, password)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if (resp.StatusCode != 302) && (resp.StatusCode/100) != 2 { //302=found or 20x reply = OK
		return fmt.Errorf("lcp server error %d", resp.StatusCode)
	}

	return nil
}
