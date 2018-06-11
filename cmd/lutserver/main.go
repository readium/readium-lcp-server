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
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/controller/lutserver"
	"github.com/readium/readium-lcp-server/lib/cron"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"io/ioutil"
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

	log.Printf("LCP server = %s", cfg.LcpServer.PublicBaseUrl)
	log.Printf("using login  %s ", cfg.LcpUpdateAuth.Username)
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
		// Cron, get license status information
		cron.Start(5 * time.Minute)
		// using Method expression instead of function
		cron.Every(1).Minutes().Do(func() {
			log.Infof("Fetching License Statuses From LSD")
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
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	server.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Printf("server is shutting down.")
	os.Exit(0)
}

func ReadLicensesPayloads(data []byte) (model.LicensesStatusCollection, error) {
	var licenses model.LicensesStatusCollection
	err := json.Unmarshal(data, &licenses)
	if err != nil {
		return nil, err
	}
	return licenses, nil
}

func FetchLicenseStatusesFromLSD(s http.IServer) {
	s.LogInfo("AUTOMATION : Fetch and save all license status documents")

	url := s.Config().LsdServer.PublicBaseUrl + "/licenses"
	auth := s.Config().LsdNotifyAuth

	// prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth.Username+":"+auth.Password)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		s.LogError("AUTOMATION : Error getting license statuses : %v", err)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.LogError("AUTOMATION : Error reading response body error : %v", err)
	}

	s.LogInfo("AUTOMATION : lsd server response : %v [http-status:%d]", body, resp.StatusCode)

	// clear the db
	err = s.Store().License().PurgeDataBase()
	if err != nil {
		panic(err)
	}

	licenses, err := ReadLicensesPayloads(body)
	if err != nil {
		panic(err)
	}
	// fill the db
	err = s.Store().License().BulkAdd(licenses)
	if err != nil {
		panic(err)
	}
}
