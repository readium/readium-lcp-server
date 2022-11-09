// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	auth "github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/readium/readium-lcp-server/config"
	frontend "github.com/readium/readium-lcp-server/frontend/server"
	"github.com/readium/readium-lcp-server/frontend/webdashboard"
	"github.com/readium/readium-lcp-server/frontend/weblicense"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/frontend/webrepository"
	"github.com/readium/readium-lcp-server/frontend/webuser"
)

func main() {
	var static, configFile string
	var err error

	if configFile = os.Getenv("READIUM_FRONTEND_CONFIG"); configFile == "" {
		configFile = "config.yaml"
	}
	config.ReadConfig(configFile)
	log.Println("Config from " + configFile)

	err = config.SetPublicUrls()
	if err != nil {
		panic(err)
	}

	//log.Println("LCP server = " + config.Config.LcpServer.PublicBaseUrl)
	//log.Println("using login  " + config.Config.LcpUpdateAuth.Username)

	driver, cnxn := config.GetDatabase(config.Config.FrontendServer.Database)
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

	repoManager, err := webrepository.Init(config.Config.FrontendServer)
	if err != nil {
		panic(err)
	}

	publicationDB, err := webpublication.Init(db)
	if err != nil {
		panic(err)
	}

	userDB, err := webuser.Open(db)
	if err != nil {
		panic(err)
	}

	purchaseDB, err := webpurchase.Init(db)
	if err != nil {
		panic(err)
	}

	dashboardDB, err := webdashboard.Init(db)
	if err != nil {
		panic(err)
	}

	licenseDB, err := weblicense.Init(db)
	if err != nil {
		panic(err)
	}

	static = config.Config.FrontendServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../frontend/manage")
	}

	filepathConfigJs := filepath.Join(static, "config.js")
	fileConfigJs, err := os.Create(filepathConfigJs)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := fileConfigJs.Close(); err != nil {
			panic(err)
		}
	}()

	configJs := `
	// This file is automatically generated, and git-ignored.
	// To ignore your local changes, use:
	// git update-index --assume-unchanged frontend/manage/config.js
	window.Config = {`
	configJs += "\n\tfrontend: {url: '" + config.Config.FrontendServer.PublicBaseUrl + "' },\n"
	configJs += "\tlcp: {url: '" + config.Config.LcpServer.PublicBaseUrl + "', user: '" + config.Config.LcpUpdateAuth.Username + "', password: '" + config.Config.LcpUpdateAuth.Password + "'},\n"
	configJs += "\tlsd: {url: '" + config.Config.LsdServer.PublicBaseUrl + "', user: '" + config.Config.LsdNotifyAuth.Username + "', password: '" + config.Config.LsdNotifyAuth.Password + "'}\n}"

	// log.Println("manage/index.html config.js:")
	// log.Println(configJs)

	fileConfigJs.WriteString(configJs)
	HandleSignals()

	// basic authentication, optional in the frontend server.
	// Authentication is used for getting user info from a license id.
	var authenticator *auth.BasicAuth
	authFile := config.Config.LsdServer.AuthFile
	if authFile != "" {
		_, err = os.Stat(authFile)
		if err != nil {
			panic(err)
		}
		htpasswd := auth.HtpasswdFileProvider(authFile)
		authenticator = auth.NewBasicAuthenticator("Basic Realm", htpasswd)
	}

	s := frontend.New(config.Config.FrontendServer.Host+":"+strconv.Itoa(config.Config.FrontendServer.Port), static, repoManager, publicationDB, userDB, dashboardDB, licenseDB, purchaseDB, authenticator)
	log.Println("Frontend webserver for LCP running on " + config.Config.FrontendServer.Host + ":" + strconv.Itoa(config.Config.FrontendServer.Port))

	if err := s.ListenAndServe(); err != nil {
		log.Println("Error " + err.Error())
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
