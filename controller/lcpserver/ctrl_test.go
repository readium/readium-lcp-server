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

package lcpserver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	goHttp "net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

// prepare test server
func TestMain(m *testing.M) {

	yamlFile, err := ioutil.ReadFile("D:\\GoProjects\\src\\readium-lcp-server\\config.yaml")
	if err != nil {
		panic(err)
	}
	var cfg http.Configuration
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		panic(err)
	}

	logz := logger.New()

	stor, err := model.SetupDB("sqlite3://file:D:\\GoProjects\\src\\readium-lcp-server\\lcp.sqlite?cache=shared&mode=rwc", logz, false)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForLCP()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}

	certFile := cfg.Certificate.Cert
	if certFile == "" {
		panic("Must specify a certificate")
	}
	privKeyFile := cfg.Certificate.PrivateKey
	if privKeyFile == "" {
		panic("Must specify a private key")
	}

	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		panic(err)
	}

	storagePath := cfg.Storage.FileSystem.Directory
	if storagePath == "" {
		storagePath = "D:\\GoProjects\\src\\readium-lcp-server\\files"
	}

	authFile := cfg.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	var s3Storage filestor.Store
	if mode := cfg.Storage.Mode; mode == "s3" {
		s3Conf := filestor.S3Config{
			ID:             cfg.Storage.AccessId,
			Secret:         cfg.Storage.Secret,
			Token:          cfg.Storage.Token,
			Endpoint:       cfg.Storage.Endpoint,
			Bucket:         cfg.Storage.Bucket,
			Region:         cfg.Storage.Region,
			DisableSSL:     cfg.Storage.DisableSSL,
			ForcePathStyle: cfg.Storage.PathStyle,
		}
		s3Storage, _ = filestor.S3(s3Conf)
	} else {
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		s3Storage = filestor.NewFileSystem(storagePath, cfg.LcpServer.PublicBaseUrl+"/files")
	}
	packager := pack.NewPackager(s3Storage, stor.Content(), 4)
	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}

	muxer := mux.NewRouter()
	server := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           ":" + strconv.Itoa(cfg.LcpServer.Port),
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Log:      logz,
		Cfg:      cfg,
		Readonly: cfg.LcpServer.ReadOnly,
		St:       &s3Storage,
		Model:    stor,
		Cert:     &cert,
		Src:      pack.ManualSource{},
	}

	server.InitAuth("Readium License Content Protection Server") // creates authority checker

	server.CreateDefaultLinks(cfg.License)

	logz.Printf("License server running on port %d [Readonly %t]", cfg.LcpServer.Port, cfg.LcpServer.ReadOnly)
	// Route.PathPrefix: http://www.gorillatoolkit.org/pkg/mux#Route.PathPrefix
	// Route.Subrouter: http://www.gorillatoolkit.org/pkg/mux#Route.Subrouter
	// Router.StrictSlash: http://www.gorillatoolkit.org/pkg/mux#Router.StrictSlash

	RegisterRoutes(muxer, server)

	server.Src.Feed(packager.Incoming)

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logz.Printf("Error " + err.Error())
		}
	}()

	m.Run()

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

func TestAddContent(t *testing.T) {

	var buf bytes.Buffer

	// generate a new uuid; this will be the content id in the lcp server
	uid, errU := uuid.NewV4()
	if errU != nil {
		t.Fatalf("Error generating UUID : %v", errU)
	}
	outputPath := "D:\\GoProjects\\src\\readium-lcp-server\\files\\sample.epub"
	inputPath := "D:\\GoProjects\\src\\readium-lcp-server\\src\\github.com\\readium\\readium-lcp-server\\test\\samples\\sample.epub"
	// encrypt the master file found at inputPath, write in the temp file, in the "encrypted repository"
	encryptedEpub, err := pack.CreateEncryptedEpub(inputPath, outputPath)

	contentDisposition := "SampleContentDisposition"

	payload := http.LcpPublication{
		ContentId:          uid.String(),
		ContentKey:         encryptedEpub.EncryptionKey,
		Output:             outputPath,
		ContentDisposition: &contentDisposition,
		Checksum:           &encryptedEpub.Checksum,
		Size:               &encryptedEpub.Size,
	}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(payload)

	req, err := http.NewRequest("PUT", "http://localhost:8081/contents/"+uid.String(), bytes.NewReader(buf.Bytes())) //Create request with JSON body

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		t.Errorf("Error : %v", err)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response body error : %v", err)
	}

	t.Logf("response : %v [http-status:%d]", string(body), resp.StatusCode)
}
