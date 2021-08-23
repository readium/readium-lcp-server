// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

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
	"strings"
	"syscall"

	auth "github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/readium/readium-lcp-server/config"
	frontend "github.com/readium/readium-lcp-server/frontend/server"
	"github.com/readium/readium-lcp-server/frontend/webdashboard"
	"github.com/readium/readium-lcp-server/frontend/weblicense"
	"github.com/readium/readium-lcp-server/frontend/webpublication"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/frontend/webrepository"
	"github.com/readium/readium-lcp-server/frontend/webuser"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var dbURI, static, configFile string
	var err error

	if configFile = os.Getenv("READIUM_FRONTEND_CONFIG"); configFile == "" {
		configFile = "config.yaml"
	}
	config.ReadConfig(configFile)
	log.Println("Read config from " + configFile)

	err = config.SetPublicUrls()
	if err != nil {
		panic(err)
	}

	log.Println("LCP server = " + config.Config.LcpServer.PublicBaseUrl)
	log.Println("using login  " + config.Config.LcpUpdateAuth.Username)

	// use a sqlite db by default
	if dbURI = config.Config.FrontendServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:frontend.sqlite?cache=shared&mode=rwc"
	}
	driver, cnxn := dbFromURI(dbURI)
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

	publicationDB, err := webpublication.Init(config.Config, db)
	if err != nil {
		panic(err)
	}

	userDB, err := webuser.Open(db)
	if err != nil {
		panic(err)
	}

	purchaseDB, err := webpurchase.Init(config.Config, db)
	if err != nil {
		panic(err)
	}

	dashboardDB, err := webdashboard.Init(config.Config, db)
	if err != nil {
		panic(err)
	}

	licenseDB, err := weblicense.Init(config.Config, db)
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

	log.Println("manage/index.html config.js:")
	log.Println(configJs)

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
	log.Println("using database " + dbURI)

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
