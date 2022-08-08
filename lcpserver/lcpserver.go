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

	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/index"
	lcpserver "github.com/readium/readium-lcp-server/lcpserver/server"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var config_file, dbURI, storagePath, certFile, privKeyFile string
	var readonly bool = false
	var err error

	if config_file = os.Getenv("READIUM_LCPSERVER_CONFIG"); config_file == "" {
		config_file = "config.yaml"
	}
	config.ReadConfig(config_file)
	log.Println("Reading config " + config_file)

	readonly = config.Config.LcpServer.ReadOnly

	err = config.SetPublicUrls()
	if err != nil {
		panic(err)
	}
	// use a sqlite db by default
	if dbURI = config.Config.LcpServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:lcp.sqlite?cache=shared&mode=rwc"
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
	idx, err := index.Open(db)
	if err != nil {
		panic(err)
	}

	lst, err := license.NewSqlStore(db)
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
		log.Println("Storage created, path", storagePath, ", URL", config.Config.Storage.FileSystem.URL)
	} else {
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
	log.Println("Using database " + dbURI)
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
