// Copyright (c) 2022 Readium Foundation
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	auth "github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/index"
	lcpserver "github.com/readium/readium-lcp-server/lcpserver/server"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
)

func main() {
	var config_file, storagePath, certFile, privKeyFile string
	var readonly bool = false
	var err error

	if config_file = os.Getenv("READIUM_LCPSERVER_CONFIG"); config_file == "" {
		config_file = "config.yaml"
	}
	config.ReadConfig(config_file)
	log.Println("Config from " + config_file)

	readonly = config.Config.LcpServer.ReadOnly

	err = config.SetPublicUrls()
	if err != nil {
		panic(err)
	}
	if certFile = config.Config.Certificate.Cert; certFile == "" {
		panic("Must specify a certificate")
	}
	if privKeyFile = config.Config.Certificate.PrivateKey; privKeyFile == "" {
		panic("Must specify a private key")
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		panic(err)
	}

	driver, cnxn := config.GetDatabase(config.Config.LcpServer.Database)
	log.Println("Database driver " + driver)

	db, err := sql.Open(driver, cnxn)
	if err != nil {
		panic(err)
	}

	if driver == "sqlite3" && !strings.Contains(cnxn, "_journal") {
		_, err = db.Exec("PRAGMA journal_mode = WAL")
		if err != nil {
			panic(err)
		}
	}

	idx, err := index.Open(db)
	if err != nil {
		panic(err)
	}

	lst, err := license.Open(db)
	if err != nil {
		panic(err)
	}

	err = license.CreateDefaultLinks()
	if err != nil {
		panic(err)
	}

	var store storage.Store
	if mode := config.Config.Storage.Mode; mode == "s3" {
		s3Conf := s3ConfigFromYAML()
		store, _ = storage.S3(s3Conf)
	} else if config.Config.Storage.FileSystem.Directory != "" {
		storagePath = config.Config.Storage.FileSystem.Directory
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		store = storage.NewFileSystem(storagePath, config.Config.Storage.FileSystem.URL)
		log.Println("Storage in", storagePath, " at URL", config.Config.Storage.FileSystem.URL)
	} else {
		store = storage.NoStorage()
		log.Println("No storage created")
	}

	packager := pack.NewPackager(store, idx, 4)

	authFile := config.Config.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}
	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}
	htpasswd := auth.HtpasswdFileProvider(authFile)
	authenticator := auth.NewBasicAuthenticator("Readium License Content Protection Server", htpasswd)

	HandleSignals()
	parsedPort := strconv.Itoa(config.Config.LcpServer.Port)
	s := lcpserver.New(":"+parsedPort, readonly, &idx, &store, &lst, &cert, packager, authenticator)
	if readonly {
		log.Println("License server running in readonly mode on port " + parsedPort)
	} else {
		log.Println("License server running on port " + parsedPort)
	}
	log.Println("Public base URL=" + config.Config.LcpServer.PublicBaseUrl)

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

func s3ConfigFromYAML() storage.S3Config {
	s3config := storage.S3Config{}

	s3config.ID = config.Config.Storage.AccessId
	s3config.Secret = config.Config.Storage.Secret
	s3config.Token = config.Config.Storage.Token

	s3config.Endpoint = config.Config.Storage.Endpoint
	s3config.Bucket = config.Config.Storage.Bucket
	s3config.Region = config.Config.Storage.Region

	s3config.DisableSSL = config.Config.Storage.DisableSSL
	s3config.ForcePathStyle = config.Config.Storage.PathStyle

	return s3config
}
