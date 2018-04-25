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
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/logger"
	"github.com/readium/readium-lcp-server/lutserver"
	"github.com/readium/readium-lcp-server/lutserver/ctrl"
	"github.com/readium/readium-lcp-server/store"
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
	cfg, err := api.ReadConfig(configFile)
	if err != nil {
		panic(err)
	}

	log.Printf("LCP server = %s", cfg.LcpServer.PublicBaseUrl)
	log.Printf("using login  %s ", cfg.LcpUpdateAuth.Username)
	// use a sqlite db by default
	if dbURI = cfg.FrontendServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:frontend.sqlite?cache=shared&mode=rwc"
	}

	stor, err := store.SetupDB(dbURI, log, true)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForFrontend()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}

	repoManager := ctrl.RepositoryManager{MasterRepositoryPath: cfg.FrontendServer.MasterRepository, EncryptedRepositoryPath: cfg.FrontendServer.EncryptedRepository}

	HandleSignals()

	server := lutserver.New(cfg, log, repoManager, stor)
	log.Printf("Frontend webserver for LCP running on " + cfg.FrontendServer.Host + ":" + strconv.Itoa(cfg.FrontendServer.Port))

	if err := server.ListenAndServe(); err != nil {
		log.Printf("Error " + err.Error())
	}
}

// HandleSignals handles system signals and adds a log before quitting
func HandleSignals() {
	sigChan := make(chan os.Signal)
	go func() {
		stacktrace := make([]byte, 1<<20)
		for sig := range sigChan {
			switch sig {
			case syscall.SIGQUIT:
				length := runtime.Stack(stacktrace, true)
				fmt.Println(string(stacktrace[:length]))
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGTERM:
				fmt.Println("Shutting down...")
				os.Exit(0)
			}
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
}
