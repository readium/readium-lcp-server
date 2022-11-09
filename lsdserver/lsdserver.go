// Copyright 2017 European Digital Reading Lab. All rights reserved.
// Licensed to the Readium Foundation under one or more contributor license agreements.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	auth "github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/readium/readium-lcp-server/config"
	licensestatuses "github.com/readium/readium-lcp-server/license_statuses"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/logging"
	lsdserver "github.com/readium/readium-lcp-server/lsdserver/server"
	"github.com/readium/readium-lcp-server/transactions"
)

func main() {
	var config_file string
	var readonly bool = false
	var err error

	if config_file = os.Getenv("READIUM_LSDSERVER_CONFIG"); config_file == "" {
		config_file = "config.yaml"
	}

	config.ReadConfig(config_file)
	log.Println("Config from " + config_file)

	err = localization.InitTranslations()
	if err != nil {
		panic(err)
	}

	readonly = config.Config.LsdServer.ReadOnly

	err = config.SetPublicUrls()
	if err != nil {
		panic(err)
	}

	driver, cnxn := config.GetDatabase(config.Config.LsdServer.Database)
	log.Println("Database driver " + driver)

	db, err := sql.Open(driver, cnxn)
	if err != nil {
		panic(err)
	}
	if driver == "sqlite3" {
		_, err = db.Exec("PRAGMA journal_mode = WAL")
		if err != nil {
			panic(err)
		}
	}

	hist, err := licensestatuses.Open(db)
	if err != nil {
		panic(err)
	}

	trns, err := transactions.Open(db)
	if err != nil {
		panic(err)
	}

	authFile := config.Config.LsdServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}

	htpasswd := auth.HtpasswdFileProvider(authFile)
	authenticator := auth.NewBasicAuthenticator("Basic Realm", htpasswd)

	// the server will behave strangely, to test the resilience of LCP compliant apps
	goofyMode := config.Config.GoofyMode

	// a log file will be created with a specifc format, for compliance testing
	complianceMode := config.Config.ComplianceMode
	logDirectory := config.Config.LsdServer.LogDirectory
	err = logging.Init(logDirectory, complianceMode)
	if err != nil {
		panic(err)
	}

	HandleSignals()

	parsedPort := strconv.Itoa(config.Config.LsdServer.Port)
	s := lsdserver.New(":"+parsedPort, readonly, complianceMode, goofyMode, &hist, &trns, authenticator)
	if readonly {
		log.Println("License status server running in readonly mode on port " + parsedPort)
	} else {
		log.Println("License status server running on port " + parsedPort)
	}
	log.Println("Public base URL=" + config.Config.LsdServer.PublicBaseUrl)

	if err := s.ListenAndServe(); err != nil {
		log.Println("Error " + err.Error())
	}

}

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
