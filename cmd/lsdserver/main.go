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
	"os"

	goHttp "net/http"
	"strconv"
	"time"

	"context"
	"os/signal"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/controller/lsdserver"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/i18n"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
)

func main() {
	logz := logger.New()
	logz.Printf("RUNNING LSD SERVER")
	// read config file
	configFile := "config.yaml"
	if os.Getenv("READIUM_LSDSERVER_CONFIG") != "" {
		configFile = os.Getenv("READIUM_LSDSERVER_CONFIG")
	}

	var err error
	cfg, err := http.ReadConfig(configFile)
	if err != nil {
		panic(err)
	}

	// check passwords file
	authFile := cfg.LsdServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	if cfg.Localization.Folder != "" {
		err = i18n.InitTranslations(cfg.Localization.Folder, cfg.Localization.Languages)
		if err != nil {
			panic(err)
		}
	}

	// use a sqlite db by default
	dbURI := "sqlite3://file:lsd.sqlite?cache=shared&mode=rwc"
	if cfg.LsdServer.Database != "" {
		dbURI = cfg.LsdServer.Database
	}

	stor, err := model.SetupDB(dbURI, logz, false)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForLSD()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}

	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}

	// a log file will be created with a specifc format, for compliance testing
	complianceMode := cfg.ComplianceMode
	logDirectory := cfg.LsdServer.LogDirectory
	err = logger.Init(logDirectory, complianceMode)
	if err != nil {
		panic(err)
	}

	parsedPort := strconv.Itoa(cfg.LsdServer.Port)
	readonly := cfg.LsdServer.ReadOnly
	// the server will behave strangely, to test the resilience of LCP compliant apps
	goofyMode := cfg.GoofyMode

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

	server := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           ":" + parsedPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Log:        logz,
		Cfg:        cfg,
		Readonly:   readonly,
		Model:      stor,
		GoophyMode: goofyMode,
	}

	server.InitAuth("Basic Realm")                   // creates authority checker
	muxer.NotFoundHandler = server.NotFoundHandler() //handle all other requests 404
	server.Log.Printf("License status server running on port %d [readonly = %t]", cfg.LsdServer.Port, readonly)

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	server.HandleFunc(muxer, licenseRoutesPathPrefix, lsdserver.FilterLicenseStatuses, true).Methods("GET")

	server.HandleFunc(licenseRoutes, "/{key}/status", lsdserver.GetLicenseStatusDocument, false).Methods("GET")

	if complianceMode {
		server.HandleFunc(muxer, "/compliancetest", lsdserver.AddLogToFile, false).Methods("POST")
	}

	server.HandleFunc(licenseRoutes, "/{key}/registered", lsdserver.ListRegisteredDevices, true).Methods("GET")
	if !readonly {
		server.HandleFunc(licenseRoutes, "/{key}/register", lsdserver.RegisterDevice, false).Methods("POST")
		server.HandleFunc(licenseRoutes, "/{key}/return", lsdserver.LendingReturn, false).Methods("PUT")
		server.HandleFunc(licenseRoutes, "/{key}/renew", lsdserver.LendingRenewal, false).Methods("PUT")
		server.HandleFunc(licenseRoutes, "/{key}/status", lsdserver.LendingCancellation, true).Methods("PATCH")

		server.HandleFunc(muxer, "/licenses", lsdserver.CreateLicenseStatusDocument, true).Methods("PUT")
		server.HandleFunc(licenseRoutes, "/", lsdserver.CreateLicenseStatusDocument, true).Methods("PUT")
	}

	logz.Printf("Using database : %q", dbURI)
	logz.Printf("Public base URL : %q", cfg.LsdServer.PublicBaseUrl)

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logz.Printf("Error " + err.Error())
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
	logz.Printf("server is shutting down.")
	os.Exit(0)
}
