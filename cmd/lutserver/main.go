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

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/controller/lutserver"
	"github.com/readium/readium-lcp-server/lib/cron"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"net"
	goHttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

func main() {
	// Start logger
	log := logger.New()
	log.Printf("RUNNING UTIL SERVER")

	var dbURI, configFile string
	var err error

	if configFile = os.Getenv("READIUM_FRONTEND_CONFIG"); configFile == "" {
		configFile = "config.yaml"
	}
	cfg, err := http.ReadConfig(configFile)
	if err != nil {
		panic(err)
	}

	// use a sqlite db by default
	if dbURI = cfg.LutServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:frontend.sqlite?cache=shared&mode=rwc"
	}

	stor, err := model.SetupDB(dbURI, log, false)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForFrontend()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}

	tcpAddress := cfg.LutServer.Host + ":" + strconv.Itoa(cfg.LutServer.Port)

	muxer := mux.NewRouter()

	muxer.Use(http.RecoveryHandler(http.RecoveryLogger(log), http.PrintRecoveryStack(true)))

	server := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           tcpAddress,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Log:   log,
		Cfg:   cfg,
		Model: stor,
	}

	// for development
	if !cfg.LutServer.StopCronRunner {
		// fetch for the first time
		go FetchLicenseStatusesFromLSD(server)
		// Cron, get license status information
		cron.Start(30 * time.Second) // TODO : set sync to 5 minutes
		// using Method expression instead of function
		cron.Every(1).Minutes().Do(func() {
			FetchLicenseStatusesFromLSD(server)
		})
	}

	lutserver.RegisterRoutes(muxer, server)

	log.Printf("Frontend webserver for LCP running on " + cfg.LutServer.Host + ":" + strconv.Itoa(cfg.LutServer.Port))

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Error " + err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	wait := time.Second * 15 // the duration for which the server gracefully wait for existing connections to finish
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// close the database
	stor.Close()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	server.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Printf("server is shutting down.")
	os.Exit(0)
}

func FetchLicenseStatusesFromLSD(s http.IServer) {
	s.LogInfo("AUTOMATION : Fetch and save all license status documents")
	lsdConn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		s.LogError("Error dialing LSD : %v\nAutomation fails.", err)
		return
	}

	defer lsdConn.Close()
	lsdRW := bufio.NewReadWriter(bufio.NewReader(lsdConn), bufio.NewWriter(lsdConn))

	_, err = lsdRW.WriteString("LICENSES\n")
	if err != nil {
		s.LogError("[LSD] Could not write : %v", err)
		return
	}

	lastSyncTime, err := s.Store().License().Latest()
	if err != nil {
		s.LogInfo("[LSD] Could not obtain last sync time: %v", err)
	}
	if lastSyncTime.IsZero() {
		s.LogInfo("[LSD] Time is zero.")
	}

	enc := gob.NewEncoder(lsdRW)
	err = enc.Encode(http.AuthorizationAndTimestamp{
		User:     s.Config().LsdNotifyAuth.Username,
		Password: s.Config().LsdNotifyAuth.Password,
		Sync:     lastSyncTime,
	})
	if err != nil {
		s.LogError("[LSD] Encode failed for struct: %v", err)
		return
	}

	err = lsdRW.Flush()
	if err != nil {
		s.LogError("[LSD] Flush failed : %v", err)
		return
	}
	// Read the reply.

	bodyBytes, err := ioutil.ReadAll(lsdRW.Reader)
	if err != nil {
		s.LogError("[LSD] Error reading response body : %v", err)
		return
	}

	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var licenses model.LicensesStatusCollection
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&licenses)
		if err != nil {
			s.LogError("[LSD] Error decoding GOB : %v\n%s", err, bodyBytes)
			return
		}
		if len(licenses) == 0 {
			s.LogInfo("No license statuses : Automation done.")
			return
		}
		licRefIds := ""
		for idx, licenseStatus := range licenses {
			if idx > 0 {
				licRefIds += ","
			}
			licRefIds += licenseStatus.LicenseRef
			s.LogInfo("LICENSE STATUS : %#v", licenseStatus)
		}

		s.LogInfo("Querying : %q", licRefIds)
		lcpLicenses := readLCPLicense(licRefIds, s)
		for _, lcpLic := range lcpLicenses {
			s.LogInfo("LCP LICENSE :  %#v", lcpLic)
		}

		// fill the db
		err = s.Store().License().BulkAddOrUpdate(licenses)
		if err != nil {
			panic(err)
		}
		s.LogInfo("Automation done.")
	} else if responseErr.Err != "" {
		s.LogError("[LSD] Replied with Error : %v", responseErr)
		return
	}

}

func readLCPLicense(uuids string, s http.IServer) model.LicensesCollection {
	lcpConn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		s.LogError("Error dialing LCP : %v\nAutomation fails.", err)
		return nil
	}

	defer lcpConn.Close()
	lcpRW := bufio.NewReadWriter(bufio.NewReader(lcpConn), bufio.NewWriter(lcpConn))
	_, err = lcpRW.WriteString("GETLICENSES\n")
	if err != nil {
		s.LogError("[LCP] Could not write : %v", err)
		return nil
	}

	enc := gob.NewEncoder(lcpRW)
	err = enc.Encode(http.AuthorizationAndLicense{
		User:     s.Config().LsdNotifyAuth.Username,
		Password: s.Config().LsdNotifyAuth.Password,
		License: &model.License{
			Id: uuids,
		},
	})
	if err != nil {
		s.LogError("[LCP] Encode failed for struct: %v", err)
		return nil
	}

	err = lcpRW.Flush()
	if err != nil {
		s.LogError("[LCP] Flush failed : %v", err)
		return nil
	}

	bodyBytes, err := ioutil.ReadAll(lcpRW.Reader)
	if err != nil {
		s.LogError("[LCP] Error reading response body : %v", err)
		return nil
	}
	var responseErr http.GobReplyError
	dec := gob.NewDecoder(bytes.NewBuffer(bodyBytes))
	err = dec.Decode(&responseErr)
	if err != nil && err != io.EOF {
		var allLicenses model.LicensesCollection
		dec = gob.NewDecoder(bytes.NewBuffer(bodyBytes))
		err = dec.Decode(&allLicenses)
		if err != nil {
			s.LogError("[LCP] Error decoding GOB : %v\n%s", err, bodyBytes)
			return nil
		}
		s.LogInfo("[LCP] Licenses :\n%v", allLicenses)
		return allLicenses
	} else {
		s.LogError("[LCP] Replied with Error : %v", responseErr)
		return nil
	}
}
