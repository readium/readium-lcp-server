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

	"github.com/abbot/go-http-auth"

	"net/http"
	"strconv"
	"time"

	"github.com/readium/readium-lcp-server/controller/common"
	"github.com/readium/readium-lcp-server/controller/lsdserver"
	"github.com/readium/readium-lcp-server/lib/localization"
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
	cfg, err := common.ReadConfig(configFile)
	if err != nil {
		panic(err)
	}

	// check passwords file
	authFile := cfg.LsdServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}
	htpasswd := auth.HtpasswdFileProvider(authFile)
	if cfg.Localization.Folder != "" {
		err = localization.InitTranslations(cfg.Localization.Folder, cfg.Localization.Languages)
		if err != nil {
			panic(err)
		}
	}

	// use a sqlite db by default
	dbURI := "sqlite3://file:lsd.sqlite?cache=shared&mode=rwc"
	if cfg.LsdServer.Database != "" {
		dbURI = cfg.LsdServer.Database
	}

	stor, err := model.SetupDB(dbURI, logz, true)
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

	common.HandleSignals()

	authenticator := auth.NewBasicAuthenticator("Basic Realm", htpasswd)
	s := New(cfg, logz, stor, authenticator)

	logz.Printf("Using database : %q", dbURI)
	logz.Printf("Public base URL : %q", cfg.LsdServer.PublicBaseUrl)

	if err = s.ListenAndServe(); err != nil {
		logz.Printf("Internal Server Error " + err.Error())
	}
}

func New(config common.Configuration,
	log logger.StdLogger,
	store model.Store,
	basicAuth *auth.BasicAuth) *common.Server {

	complianceMode := config.ComplianceMode
	parsedPort := strconv.Itoa(config.LsdServer.Port)
	readonly := config.LsdServer.ReadOnly
	// the server will behave strangely, to test the resilience of LCP compliant apps
	goofyMode := config.GoofyMode

	sr := common.CreateServerRouter("")

	s := &common.Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           ":" + parsedPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Log:        log,
		Cfg:        config,
		Readonly:   readonly,
		ORM:        store,
		GoophyMode: goofyMode,
	}

	s.Log.Printf("License status server running on port %d [readonly = %t]", config.LsdServer.Port, readonly)

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.HandlePrivateFunc(sr.R, licenseRoutesPathPrefix, lsdserver.FilterLicenseStatuses, basicAuth).Methods("GET")

	s.HandleFunc(licenseRoutes, "/{key}/status", lsdserver.GetLicenseStatusDocument).Methods("GET")

	if complianceMode {
		s.HandleFunc(sr.R, "/compliancetest", lsdserver.AddLogToFile).Methods("POST")
	}

	s.HandlePrivateFunc(licenseRoutes, "/{key}/registered", lsdserver.ListRegisteredDevices, basicAuth).Methods("GET")
	if !readonly {
		s.HandleFunc(licenseRoutes, "/{key}/register", lsdserver.RegisterDevice).Methods("POST")
		s.HandleFunc(licenseRoutes, "/{key}/return", lsdserver.LendingReturn).Methods("PUT")
		s.HandleFunc(licenseRoutes, "/{key}/renew", lsdserver.LendingRenewal).Methods("PUT")
		s.HandlePrivateFunc(licenseRoutes, "/{key}/status", lsdserver.LendingCancellation, basicAuth).Methods("PATCH")

		s.HandlePrivateFunc(sr.R, "/licenses", lsdserver.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
		s.HandlePrivateFunc(licenseRoutes, "/", lsdserver.CreateLicenseStatusDocument, basicAuth).Methods("PUT")
	}

	return s
}
