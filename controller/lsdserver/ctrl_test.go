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

package lsdserver

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	goHttp "net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
)

var workingDir string
var localhostAndPort string

// debugging multiple header calls
type debugLogger struct{}

func (d debugLogger) Write(p []byte) (n int, err error) {
	s := string(p)
	if strings.Contains(s, "multiple response.WriteHeader") {
		debug.PrintStack()
	}
	return os.Stderr.Write(p)
}

// prepare test server
func TestMain(m *testing.M) {
	var err error
	logz := logger.New()
	// working dir
	workingDir, err = os.Getwd()
	if err != nil {
		panic("Working dir error : " + err.Error())
	}
	workingDir = strings.Replace(workingDir, "\\src\\github.com\\readium\\readium-lcp-server\\controller\\lsdserver", "", -1)

	yamlFile, err := ioutil.ReadFile(workingDir + "\\config.yaml")
	if err != nil {
		panic(err)
	}
	var cfg http.Configuration
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		panic(err)
	}

	stor, err := model.SetupDB("sqlite3://file:"+workingDir+"\\lsd.sqlite?cache=shared&mode=rwc", logz, false)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForLSD()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}
	// a log file will be created with a specifc format, for compliance testing
	err = logger.Init(cfg.LsdServer.LogDirectory, cfg.ComplianceMode)
	if err != nil {
		panic(err)
	}

	storagePath := cfg.Storage.FileSystem.Directory
	if storagePath == "" {
		storagePath = workingDir + "\\files"
	}

	authFile := cfg.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	muxer := mux.NewRouter()
	muxer.Use(
		http.RecoveryHandler(http.RecoveryLogger(logz), http.PrintRecoveryStack(true)),
		http.CorsMiddleWare(
			http.AllowedOrigins([]string{"*"}),
			http.AllowedMethods([]string{"PATCH", "HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE"}),
			http.AllowedHeaders([]string{"Range", "Content-Type", "Origin", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Authorization"}),
		),
		http.DelayMiddleware,
	)
	runningPort := strconv.Itoa(cfg.LcpServer.Port)
	localhostAndPort = "http://localhost:" + runningPort
	server := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           ":" + runningPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(debugLogger{}, "", 0), // debugging multiple header calls
		},
		Log:        logz,
		Cfg:        cfg,
		Readonly:   cfg.LcpServer.ReadOnly,
		Model:      stor,
		GoophyMode: cfg.GoofyMode,
	}

	server.InitAuth("Basic Realm") // creates authority checker

	logz.Printf("License status server running on port %d [Readonly %t]", cfg.LcpServer.Port, cfg.LcpServer.ReadOnly)

	RegisterRoutes(muxer, server)

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logz.Printf("ListenAndServe Error " + err.Error())
		}
	}()

	m.Run()

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
	logz.Printf("server is shutting down.")
	os.Exit(0)
}
