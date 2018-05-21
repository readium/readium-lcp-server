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
	"runtime"

	"crypto/tls"

	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/readium/readium-lcp-server/controller/common"
	"github.com/readium/readium-lcp-server/controller/lcpserver"
	"github.com/readium/readium-lcp-server/lib/file_storage"
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
	cfg, err := common.ReadConfig(configFile)
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

	var s3Storage file_storage.Store
	if mode := cfg.Storage.Mode; mode == "s3" {
		s3Conf := s3ConfigFromYAML(cfg.Storage)
		s3Storage, _ = file_storage.S3(s3Conf)
	} else {
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		s3Storage = file_storage.NewFileSystem(storagePath, cfg.LcpServer.PublicBaseUrl+"/files")
	}
	// Prepare packager with S3 and storage
	packager := pack.NewPackager(s3Storage, stor.Content(), 4)
	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}
	//
	// finally, starting server
	common.HandleSignals()
	s := New(cfg, logz, stor, &s3Storage, &cert, packager)

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

func New(
	cfg common.Configuration,
	log logger.StdLogger,
	stor model.Store,
	storage *file_storage.Store,
	cert *tls.Certificate,
	packager *pack.Packager) *common.Server {

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

	sr := common.CreateServerRouter(static)

	s := &common.Server{
		Server: http.Server{
			Handler:        sr.N,
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

	s.MakeAutorizator("Readium License Content Protection Server") // creates authority checker
	sr.R.NotFoundHandler = s.NotFoundHandler()                     //handle all other requests 404
	s.CreateDefaultLinks(cfg.License)

	log.Printf("License server running on port %d [Readonly %t]", cfg.LcpServer.Port, readonly)
	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	// methods related to EPUB encrypted content

	contentRoutesPathPrefix := "/contents"
	contentRoutes := sr.R.PathPrefix(contentRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.HandleFunc(sr.R, contentRoutesPathPrefix, lcpserver.ListContents).Methods("GET")

	// get encrypted content by content id (a uuid)
	s.HandleFunc(contentRoutes, "/{content_id}", lcpserver.GetContent).Methods("GET")
	// get all licenses associated with a given content
	s.HandlePrivateFunc(contentRoutes, "/{content_id}/licenses", lcpserver.ListLicensesForContent).Methods("GET")

	if !readonly {
		s.HandleFunc(contentRoutes, "/{name}", lcpserver.StoreContent).Methods("POST")
		// put content to the storage
		s.HandlePrivateFunc(contentRoutes, "/{content_id}", lcpserver.AddContent).Methods("PUT")
		// generate a license for given content
		s.HandlePrivateFunc(contentRoutes, "/{content_id}/license", lcpserver.GenerateLicense).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.HandlePrivateFunc(contentRoutes, "/{content_id}/licenses", lcpserver.GenerateLicense).Methods("POST")
		// generate a licensed publication
		s.HandlePrivateFunc(contentRoutes, "/{content_id}/publication", lcpserver.GenerateLicensedPublication).Methods("POST")
		// deprecated, from a typo in the lcp server spec
		s.HandlePrivateFunc(contentRoutes, "/{content_id}/publications", lcpserver.GenerateLicensedPublication).Methods("POST")
	}

	// methods related to licenses

	licenseRoutesPathPrefix := "/licenses"
	licenseRoutes := sr.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)

	s.HandlePrivateFunc(sr.R, licenseRoutesPathPrefix, lcpserver.ListLicenses).Methods("GET")
	// get a license
	s.HandlePrivateFunc(licenseRoutes, "/{license_id}", lcpserver.GetLicense).Methods("GET")
	s.HandlePrivateFunc(licenseRoutes, "/{license_id}", lcpserver.GetLicense).Methods("POST")
	// get a licensed publication via a license id
	s.HandlePrivateFunc(licenseRoutes, "/{license_id}/publication", lcpserver.GetLicensedPublication).Methods("POST")
	if !readonly {
		// update a license
		s.HandlePrivateFunc(licenseRoutes, "/{license_id}", lcpserver.UpdateLicense).Methods("PATCH")
	}

	s.Src.Feed(packager.Incoming)
	return s
}

func s3ConfigFromYAML(cfg common.Storage) file_storage.S3Config {
	s3config := file_storage.S3Config{
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
