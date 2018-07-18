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

package lutserver

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"testing"
)

func GOBClient(t *testing.T, addr string) (*bufio.ReadWriter, net.Conn, error) {
	// Dial the remote process.
	// Note that the local port is chosen on the fly. If the local port
	// must be a specific one, use DialTCP() instead.
	t.Logf("Dial " + addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("Dialing %s failed : %v", addr, err)
	}
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), conn, nil
}

/*
## The client and server functions

With all this in place, we can now set up client and server functions.

The client function connects to the server and sends STRING and GOB requests.

The server starts listening for requests and triggers the appropriate handlers.
*/

// client is called if the app is called with -connect=`ip addr`.
func client(t *testing.T, ip string) error {

	// A struct with a mix of fields, used for the GOB example.
	type complexData struct {
		N int
		S string
		M map[string]int
		P []byte
		C *complexData
	}
	// Some test data. Note how GOB even handles maps, slices, and
	// recursive data structures without problems.
	testStruct := complexData{
		N: 23,
		S: "string data",
		M: map[string]int{"one": 1, "two": 2, "three": 3},
		P: []byte("abc"),
		C: &complexData{
			N: 256,
			S: "Recursive structs? Piece of cake!",
			M: map[string]int{"01": 1, "10": 2, "11": 3},
		},
	}

	// Open a connection to the server.
	rw, cl, err := GOBClient(t, ip+":9000")
	if err != nil {
		return fmt.Errorf("Client: Failed to open connection to %v:9000 -> %v", ip, err)
	}
	defer cl.Close()
	// Send a STRING request.
	// Send the request name.
	// Send the data.
	log.Println("Send the string request.")
	n, err := rw.WriteString("STRING\n")
	if err != nil {
		return fmt.Errorf("Could not send the STRING request (%v bytes written) : %v", strconv.Itoa(n), err)
	}
	n, err = rw.WriteString("Additional data.\n")
	if err != nil {
		return fmt.Errorf("Could not send additional STRING data %v bytes written) : %v", strconv.Itoa(n), err)
	}
	log.Println("Flush the buffer.")
	err = rw.Flush()
	if err != nil {
		return fmt.Errorf("Flush failed : %v", err)
	}

	// Read the reply.
	log.Println("Read the reply.")
	response, err := rw.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Client: Failed to read the reply: %v : %v", response, err)
	}

	log.Println("STRING request: got a response:", response)

	// Send a GOB request.
	// Create an encoder that directly transmits to `rw`.
	// Send the request name.
	// Send the GOB.
	log.Println("Send a struct as GOB:")
	log.Printf("Outer complexData struct: \n%#v\n", testStruct)
	log.Printf("Inner complexData struct: \n%#v\n", testStruct.C)
	enc := gob.NewEncoder(rw)
	n, err = rw.WriteString("GOB\n")
	if err != nil {
		return fmt.Errorf("Could not write GOB data (%v) bytes written) : %v", strconv.Itoa(n), err)
	}
	err = enc.Encode(testStruct)
	if err != nil {
		return fmt.Errorf("Encode failed for struct: %#v : %v", testStruct, err)
	}
	err = rw.Flush()
	if err != nil {
		return fmt.Errorf("Flush failed : %v", err)
	}
	return nil
}

// server listens for incoming requests and dispatches them to
// registered handler functions.
func server() (*http.GobEndpoint, error) {
	logz := logger.New()
	endpoint := http.NewGobEndpoint(logz)

	// Add the handle funcs.
	endpoint.AddHandleFunc("STRING", handleStrings)
	endpoint.AddHandleFunc("GOB", handleGob)

	// Start listening.
	return endpoint, endpoint.Listen(":9000")
}

/* Now let's create two handler functions. The easiest case is where our
ad-hoc protocol only sends string data.

The second handler receives and processes a struct that was send as GOB data.
*/

// handleStrings handles the "STRING" request.
func handleStrings(rw *bufio.ReadWriter) error {
	// Receive a string.
	log.Print("Receive STRING message:")
	s, err := rw.ReadString('\n')
	if err != nil {
		log.Println("Cannot read from connection.\n", err)
	}
	s = strings.Trim(s, "\n ")
	log.Println(s)
	_, err = rw.WriteString("Thank you.\n")
	if err != nil {
		log.Println("Cannot write to connection.\n", err)
	}
	err = rw.Flush()
	if err != nil {
		log.Println("Flush failed.", err)
	}
	return nil
}

// handleGob handles the "GOB" request. It decodes the received GOB data
// into a struct.
func handleGob(rw *bufio.ReadWriter) error {
	// A struct with a mix of fields, used for the GOB example.
	type complexData struct {
		N int
		S string
		M map[string]int
		P []byte
		C *complexData
	}
	log.Print("Receive GOB data:")
	var data complexData
	// Create a decoder that decodes directly into a struct variable.
	dec := gob.NewDecoder(rw)
	err := dec.Decode(&data)
	if err != nil {
		return fmt.Errorf("Error decoding GOB : %v", err)
	}
	// Print the complexData struct and the nested one, too, to prove
	// that both travelled across the wire.
	log.Printf("Outer complexData struct: \n%#v\n", data)
	log.Printf("Inner complexData struct: \n%#v\n", data.C)
	return nil
}

func TestInterserverLocal(t *testing.T) {
	t.Log("Starting test.")
	go func() {
		t.Log("Starting server")
		_, err := server()
		if err != nil {
			t.Fatalf("Error : %v", err)
		}
	}()

	t.Log("Testing.")
	err := client(t, "localhost")
	if err != nil {
		t.Fatalf("Error : %v", err)
	}
	t.Log("Client done.")

}

func TestInterserver(t *testing.T) {
	// Open a connection to the server.
	rw, cl, err := GOBClient(t, "localhost:9000")
	if err != nil {
		t.Fatalf("Client: Failed to open connection  : %v", err)
	}

	defer cl.Close()
	t.Log("Placing a licenses request.")
	_, err = rw.WriteString("LICENSES\n")
	if err != nil {
		t.Fatalf("Could not write : %v", err)
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(http.Authorization{User: "badu", Password: "hello"})
	if err != nil {
		t.Fatalf("Encode failed for struct: %v", err)
	}

	t.Log("Flushing the command.")
	err = rw.Flush()
	if err != nil {
		t.Fatalf("Flush failed : %v", err)
	}
	// Read the reply.
	t.Log("Read the reply.")

	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		t.Fatalf("Error reading response body : %v", err)
	}

	var data model.LicensesStatusCollection
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&data)
	if err != nil {
		t.Fatalf("Error decoding GOB data: %v\n%s", err, bodyBytes)
	}
	t.Logf("Response (%d statuses):\n%v", len(data), data)
}

func TestInterserver2(t *testing.T) {
	// Open a connection to the server.
	rw, cl, err := GOBClient(t, "localhost:9000")
	if err != nil {
		t.Fatalf("Client: Failed to open connection  : %v", err)
	}

	defer cl.Close()
	t.Log("Placing a license request.")
	_, err = rw.WriteString("LICENSESTATUS\n")
	if err != nil {
		t.Fatalf("Could not write : %v", err)
	}

	enc := gob.NewEncoder(rw)
	err = enc.Encode(http.AuthorizationAndLicense{User: "badu", Password: "hello", License: &model.License{Id: "5d14fcd1-f4ec-43e9-b1cf-3b163c2155d8"}})
	if err != nil {
		t.Fatalf("Encode failed for struct: %v", err)
	}

	t.Log("Flushing the command.")
	err = rw.Flush()
	if err != nil {
		t.Fatalf("Flush failed : %v", err)
	}
	// Read the reply.
	t.Log("Read the reply.")

	bodyBytes, err := ioutil.ReadAll(rw.Reader)
	if err != nil {
		t.Fatalf("Error reading response body : %v", err)
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var license model.LicenseStatus
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&license)
		if err != nil {
			t.Errorf("[LSD] Error decoding GOB : %v\n%s", err, bodyBytes)
		}
		t.Logf("License : %#v", license)

	} else if responseErr.Err != "" {
		t.Errorf("[LSD] Replied with Error : %v", responseErr)
	}
}
