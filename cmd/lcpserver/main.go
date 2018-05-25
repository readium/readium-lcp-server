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
	"context"
	"crypto/tls"
	"os"
	"runtime"

	goHttp "net/http"
	"path/filepath"
	"strconv"
	"time"

	"os/signal"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/controller/lcpserver"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
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
	cfg, err := http.ReadConfig(configFile)
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
	stor, err := model.SetupDB(dbURI, logz, false)
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

	var s3Storage filestor.Store
	if mode := cfg.Storage.Mode; mode == "s3" {
		s3Conf := s3ConfigFromYAML(cfg.Storage)
		s3Storage, _ = filestor.S3(s3Conf)
	} else {
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		s3Storage = filestor.NewFileSystem(storagePath, cfg.LcpServer.PublicBaseUrl+"/files")
	}
	// Prepare packager with S3 and storage
	packager := pack.NewPackager(s3Storage, stor.Content(), 4)
	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}
	//
	// finally, starting server
	server := New(cfg, logz, stor, &s3Storage, &cert, packager)

	logz.Printf("Using database " + dbURI)
	logz.Printf("Public base URL=" + cfg.LcpServer.PublicBaseUrl)
	logz.Printf("License links:")
	for nameOfLink, link := range cfg.License.Links {
		logz.Printf("  " + nameOfLink + " => " + link)
	}

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

func New(
	cfg http.Configuration,
	log logger.StdLogger,
	stor model.Store,
	storage *filestor.Store,
	cert *tls.Certificate,
	packager *pack.Packager) *http.Server {

	parsedPort := strconv.Itoa(cfg.LcpServer.Port)

	readonly := cfg.LcpServer.ReadOnly

	// writing static
	static := cfg.LcpServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../../web/lcp")
	}
	filepathConfigJs := filepath.Join(static, "/config.js")
	fileConfigJs, err := os.Create(filepathConfigJs)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fileConfigJs.Close(); err != nil {
			panic(err)
		}
	}()

	static = cfg.LcpServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../lcpserver/manage")
	}
	configJs := "// This file is automatically generated, and git-ignored.\n// To ignore your local changes, use:\n// git update-index --assume-unchanged lcpserver/manage/config.js\n\nvar Config = {\n    lcp: {url: '" + cfg.LcpServer.PublicBaseUrl + "', user:'" + cfg.LcpUpdateAuth.Username + "', password: '" + cfg.LcpUpdateAuth.Password + "'},\n    lsd: {url: '" + cfg.LsdServer.PublicBaseUrl + "', user:'" + cfg.LcpUpdateAuth.Username + "', password: '" + cfg.LcpUpdateAuth.Password + "'}\n}\n"

	log.Printf("manage/index.html config.js:")
	log.Printf(configJs)
	fileConfigJs.WriteString(configJs)

	muxer := mux.NewRouter()

	muxer.Use(
		http.RecoveryHandler(http.RecoveryLogger(log), http.PrintRecoveryStack(true)),
		http.CorsMiddleWare(
			http.AllowedOrigins([]string{"*"}),
			http.AllowedMethods([]string{"PATCH", "HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE"}),
			http.AllowedHeaders([]string{"Range", "Content-Type", "Origin", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Authorization"}),
		),
		http.DelayMiddleware,
	)

	if static != "" {
		// TODO : fix this.
		muxer.PathPrefix("/static/").Handler(goHttp.StripPrefix("/static/", goHttp.FileServer(goHttp.Dir(static))))
	}

	s := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           ":" + parsedPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Log:      log,
		Cfg:      cfg,
		Readonly: readonly,
		St:       storage,
		Model:    stor,
		Cert:     cert,
		Src:      pack.ManualSource{},
	}

	s.InitAuth("Readium License Content Protection Server") // creates authority checker
	muxer.NotFoundHandler = s.NotFoundHandler()             // handle all other requests 404

	s.CreateDefaultLinks(cfg.License)

	log.Printf("License server running on port %d [Readonly %t]", cfg.LcpServer.Port, readonly)
	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	// methods related to EPUB encrypted content

	contentRoutesPathPrefix := "/contents"
	contentRoutes := muxer.PathPrefix(contentRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.HandleFunc(muxer, contentRoutesPathPrefix, lcpserver.ListContents, false).Methods("GET")

	// get encrypted content by content id (a uuid)
	s.HandleFunc(contentRoutes, "/{content_id}", lcpserver.GetContent, false).Methods("GET")
	// get all licenses associated with a given content
	s.HandleFunc(contentRoutes, "/{content_id}/licenses", lcpserver.ListLicensesForContent, true).Methods("GET")

	if !readonly {
		s.HandleFunc(contentRoutes, "/{name}", lcpserver.StoreContent, true).Methods("POST")
		// put content to the storage
		s.HandleFunc(contentRoutes, "/{content_id}", lcpserver.AddContent, true).Methods("PUT")
		// generate a license for given content
		s.HandleFunc(contentRoutes, "/{content_id}/license", lcpserver.GenerateLicense, true).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.HandleFunc(contentRoutes, "/{content_id}/licenses", lcpserver.GenerateLicense, true).Methods("POST")
		// generate a licensed publication
		s.HandleFunc(contentRoutes, "/{content_id}/publication", lcpserver.GenerateLicensedPublication, true).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.HandleFunc(contentRoutes, "/{content_id}/publications", lcpserver.GenerateLicensedPublication, true).Methods("POST")
	}

	// methods related to licenses

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.HandleFunc(muxer, licenseRoutesPathPrefix, lcpserver.ListLicenses, true).Methods("GET")
	// get a license
	s.HandleFunc(licenseRoutes, "/{license_id}", lcpserver.GetLicense, true).Methods("GET")
	s.HandleFunc(licenseRoutes, "/{license_id}", lcpserver.GetLicense, true).Methods("POST")
	// get a licensed publication via a license id
	s.HandleFunc(licenseRoutes, "/{license_id}/publication", lcpserver.GetLicensedPublication, true).Methods("POST")
	if !readonly {
		// update a license
		s.HandleFunc(licenseRoutes, "/{license_id}", lcpserver.UpdateLicense, true).Methods("PATCH")
	}

	s.Src.Feed(packager.Incoming)
	return s
}

func s3ConfigFromYAML(cfg http.Storage) filestor.S3Config {
	s3config := filestor.S3Config{
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
