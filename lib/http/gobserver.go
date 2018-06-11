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

package http

import (
	"bufio"
	"fmt"
	"github.com/readium/readium-lcp-server/lib/logger"
	"io"
	"net"
	"strings"
	"time"
)

// NewGobEndpoint creates a new endpoint. Too keep things simple,
// the endpoint listens on a fixed port number.
func NewGobEndpoint(log logger.StdLogger) *GobEndpoint {
	// Create a new Endpoint with an empty list of handler funcs.
	return &GobEndpoint{
		handler: map[string]GobHandleFunc{}, log: log,
	}
}

// AddHandleFunc adds a new function for handling incoming data.
func (e *GobEndpoint) AddHandleFunc(name string, f GobHandleFunc) {
	e.m.Lock()
	e.handler[name] = f
	e.m.Unlock()
}

// Listen starts listening on the endpoint port on all interfaces.
// At least one handler function must have been added
// through AddHandleFunc() before.
func (e *GobEndpoint) Listen(port string) error {
	var err error
	e.listener, err = net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("Unable to listen on %s : %v\n", e.listener.Addr().String(), err)
	}
	defer e.listener.Close()
	e.log.Printf("Listening GOB requests on %s", e.listener.Addr().String())
	for {
		//e.log.Printf("Accept a connection request.")
		conn, err := e.listener.Accept()
		if err != nil {
			e.log.Printf("Failed accepting a connection request:", err)
			continue
		}
		//e.log.Printf("Handle incoming messages.")
		go e.handleMessages(conn)
	}
}

// handleMessages reads the connection up to the first newline.
// Based on this string, it calls the appropriate GobHandleFunc.
func (e *GobEndpoint) handleMessages(conn net.Conn) {
	// Wrap the connection into a buffered reader for easier reading.
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))

	defer conn.Close()

	// Read from the connection until EOF. Expect a command name as the
	// next input. Call the handler that is registered for this command.
	for {
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			//e.log.Println("Reached EOF - closing this connection.\n   ---")
			return
		case err != nil:
			//e.log.Println("\nError reading command. Got: '"+cmd+"'\n", err)
			return
		}
		// Trim the request string - ReadString does not strip any newlines.
		cmd = strings.Trim(cmd, "\n ")
		//e.log.Printf("Received command %q", cmd)

		// Fetch the appropriate handler function from the 'handler' map and call it.
		e.m.RLock()
		handleCommand, ok := e.handler[cmd]
		e.m.RUnlock()
		if !ok {
			e.log.Errorf("Command %q is not registered.", cmd)
			return
		}
		err = handleCommand(rw)
		if err != nil {
			e.log.Errorf("Handler returned error : %v", err)
			rw.WriteString(err.Error())
		}

		err = rw.Flush()
		if err != nil {
			e.log.Printf("Flush failed : %v", err)
		}
	}
}
