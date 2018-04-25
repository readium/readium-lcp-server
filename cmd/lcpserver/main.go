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
	"syscall"

	"github.com/abbot/go-http-auth"

	"crypto/tls"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/lcpserver"
	"github.com/readium/readium-lcp-server/logger"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
	"github.com/readium/readium-lcp-server/store"
)

func main() {
	logz := logger.New()

	var storagePath, certFile, privKeyFile string
	var err error
	logz.Printf("RUNNING LCP SERVER")
	configFile := "config.yaml"
	if os.Getenv("READIUM_LCPSERVER_CONFIG") != "" {
		configFile = os.Getenv("READIUM_LCPSERVER_CONFIG")
	}

	logz.Printf("Reading config " + configFile)
	cfg, err := api.ReadConfig(configFile)
	if err != nil {
		panic(err)
	}
	if certFile = cfg.Certificate.Cert; certFile == "" {
		panic("Must specify a certificate")
	}
	if privKeyFile = cfg.Certificate.PrivateKey; privKeyFile == "" {
		panic("Must specify a private key")
	}
	authFile := cfg.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		panic(err)
	}

	// use a sqlite db by default
	dbURI := "sqlite3://file:lcp.sqlite?cache=shared&mode=rwc"
	if cfg.LcpServer.Database != "" {
		dbURI = cfg.LcpServer.Database
	}

	// Init database
	stor, err := store.SetupDB(dbURI, logz, true)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForLCP()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}

	if storagePath = cfg.Storage.FileSystem.Directory; storagePath == "" {
		storagePath = "files"
	}

	var s3Storage storage.Store
	if mode := cfg.Storage.Mode; mode == "s3" {
		s3Conf := s3ConfigFromYAML(cfg.Storage)
		s3Storage, _ = storage.S3(s3Conf)
	} else {
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		s3Storage = storage.NewFileSystem(storagePath, cfg.LcpServer.PublicBaseUrl+"/files")
	}
	// Prepare packager with S3 and storage
	packager := pack.NewPackager(s3Storage, stor.Content(), 4)
	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}
	//
	htpasswd := auth.HtpasswdFileProvider(authFile)
	authenticator := auth.NewBasicAuthenticator("Readium License Content Protection Server", htpasswd)

	// finally, starting server
	HandleSignals()
	s := lcpserver.New(cfg, logz, stor, &s3Storage, &cert, packager, authenticator)

	logz.Printf("Using database " + dbURI)
	logz.Printf("Public base URL=" + cfg.LcpServer.PublicBaseUrl)
	logz.Printf("License links:")
	for nameOfLink, link := range cfg.License.Links {
		logz.Printf("  " + nameOfLink + " => " + link)
	}

	if err = s.ListenAndServe(); err != nil {
		logz.Printf("Internal Server Error " + err.Error())
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

func s3ConfigFromYAML(cfg api.Storage) storage.S3Config {
	s3config := storage.S3Config{
		ID:             cfg.AccessId,
		Secret:         cfg.Secret,
		Token:          cfg.Token,
		Endpoint:       cfg.Endpoint,
		Bucket:         cfg.Bucket,
		Region:         cfg.Region,
		DisableSSL:     cfg.DisableSSL,
		ForcePathStyle: cfg.PathStyle,
	}

	return s3config
}
