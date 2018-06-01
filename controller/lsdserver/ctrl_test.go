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

package lsdserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"log"
	goHttp "net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
)

var workingDir string
var localhostAndPort string

// debugging multiple header calls
type debugLogger struct{}

func (d debugLogger) Write(p []byte) (n int, err error) {
	s := string(p)
	if strings.Contains(s, "multiple response.WriteHeader") {
		debug.PrintStack()
	}
	return os.Stderr.Write(p)
}

// prepare test server
func TestMain(m *testing.M) {
	var err error
	logz := logger.New()
	// working dir
	workingDir, err = os.Getwd()
	if err != nil {
		panic("Working dir error : " + err.Error())
	}
	workingDir = strings.Replace(workingDir, "\\src\\github.com\\readium\\readium-lcp-server\\controller\\lsdserver", "", -1)

	yamlFile, err := ioutil.ReadFile(workingDir + "\\config.yaml")
	if err != nil {
		panic(err)
	}
	var cfg http.Configuration
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		panic(err)
	}

	stor, err := model.SetupDB("sqlite3://file:"+workingDir+"\\lsd.sqlite?cache=shared&mode=rwc", logz, true) // in debug mode, 'cause we're testing
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	err = stor.AutomigrateForLSD()
	if err != nil {
		panic("Error migrating database : " + err.Error())
	}
	// a log file will be created with a specifc format, for compliance testing
	err = logger.Init(cfg.LsdServer.LogDirectory, cfg.ComplianceMode)
	if err != nil {
		panic(err)
	}

	storagePath := cfg.Storage.FileSystem.Directory
	if storagePath == "" {
		storagePath = workingDir + "\\files"
	}

	authFile := cfg.LsdServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	muxer := mux.NewRouter()
	muxer.Use(
		http.RecoveryHandler(http.RecoveryLogger(logz), http.PrintRecoveryStack(true)),
		http.CorsMiddleWare(
			http.AllowedOrigins([]string{"*"}),
			http.AllowedMethods([]string{"PATCH", "HEAD", "POST", "GET", "OPTIONS", "PUT", "DELETE"}),
			http.AllowedHeaders([]string{"Range", "Content-Type", "Origin", "X-Requested-With", "Accept", "Accept-Language", "Content-Language", "Authorization"}),
		),
		http.DelayMiddleware,
	)
	runningPort := strconv.Itoa(cfg.LsdServer.Port)
	localhostAndPort = "http://localhost:" + runningPort
	server := &http.Server{
		Server: goHttp.Server{
			Handler:        muxer,
			Addr:           ":" + runningPort,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
			ErrorLog:       log.New(debugLogger{}, "", 0), // debugging multiple header calls
		},
		Log:        logz,
		Cfg:        cfg,
		Readonly:   cfg.LsdServer.ReadOnly,
		Model:      stor,
		GoophyMode: cfg.GoofyMode,
	}

	server.InitAuth("Basic Realm", cfg.LsdServer.AuthFile) // creates authority checker

	logz.Printf("License status server running on port %d [Readonly %t]", cfg.LsdServer.Port, cfg.LsdServer.ReadOnly)

	RegisterRoutes(muxer, server)

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil {
			logz.Printf("ListenAndServe Error " + err.Error())
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

func TestAddLogToFile(t *testing.T) {
	req, err := http.NewRequest("POST", localhostAndPort+"/compliancetest?test_stage=start&test_number=3&test_result=s", nil)
	//req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))

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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	}
}

func TestCreateLicenseStatusDocument(t *testing.T) {
	var buf bytes.Buffer

	uuid, _ := model.NewUUID()
	payload := model.License{
		Id:        uuid.String(),
		ContentId: uuid.String(),
		Provider:  "Google",
		User: &model.User{
			UUID: uuid.String(),
		},
		Encryption: model.LicenseEncryption{
			UserKey: model.LicenseUserKey{
				Hint:  "Hint",
				Value: "PasswordPassword",
			},
		},
		Rights: &model.LicenseUserRights{
			Start: &model.NullTime{
				Valid: true,
				Time:  time.Now(),
			},
			End: &model.NullTime{
				Valid: true,
				Time:  time.Now().Add(30 * 24 * time.Hour),
			},
		},
	}

	enc := json.NewEncoder(&buf)
	enc.Encode(payload)

	req, err := http.NewRequest("PUT", localhostAndPort+"/licenses", bytes.NewReader(buf.Bytes()))
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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	}
}

func TestGetLicenseStatusDocument(t *testing.T) {
	req, err := http.NewRequest("GET", localhostAndPort+"/licenses/673ffd51-3485-40bf-a246-7d35e8e163c4/status", nil)
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept-Language", "Ro_ro")
	req.Header.Set("Ignored", "Ignored header")
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

	if resp.StatusCode >= 300 {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	} else {
		t.Logf("Raw response : %s", string(body))
		var payload model.LicenseStatus
		err = json.Unmarshal(body, &payload)
		if err != nil {
			t.Fatalf("Error unmarshaling : %v", err)
		}
		t.Logf("Response : %#v", payload)
	}
}

func TestFilterLicenseStatuses(t *testing.T) {
	req, err := http.NewRequest("GET", localhostAndPort+"/licenses?devices=3&page=1&per_page=2", nil)
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

	if resp.StatusCode >= 300 {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	} else {
		t.Logf("Raw response : %s", string(body))
		var payload model.LicensesStatusCollection
		err = json.Unmarshal(body, &payload)
		if err != nil {
			t.Fatalf("Error unmarshaling : %v", err)
		}
		t.Logf("Response : %#v", payload)
	}
}

func TestListRegisteredDevices(t *testing.T) {
	req, err := http.NewRequest("GET", localhostAndPort+"/licenses/08d7cf49-d3b2-4183-9971-66ceb1636f8e/registered", nil)
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

	if resp.StatusCode >= 300 {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	} else {
		t.Logf("Raw response : %s", string(body))
		var payload model.TransactionEventsCollection
		err = json.Unmarshal(body, &payload)
		if err != nil {
			t.Fatalf("Error unmarshaling : %v", err)
		}
		t.Logf("Response : %#v", payload)
	}
}

func TestRegisterDevice(t *testing.T) {

	req, err := http.NewRequest("POST", localhostAndPort+"/licenses/08d7cf49-d3b2-4183-9971-66ceb1636f8e/register?name=TESTDEVICE&id=9e29aa7c-9105-42ad-b344-c0c4bfbaa529&end=today", nil)
	//req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept-Language", "Ro_ro")
	req.Header.Set("Ignored", "Ignored header")
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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	}
}

func TestLendingReturn(t *testing.T) {
	req, err := http.NewRequest("PUT", localhostAndPort+"/licenses/08d7cf49-d3b2-4183-9971-66ceb1636f8e/return?name=TESTDEVICE&id=9e29aa7c-9105-42ad-b344-c0c4bfbaa529&end=today", nil)
	//req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept-Language", "Ro_ro")
	req.Header.Set("Ignored", "Ignored header")
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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	}
}

func TestLendingRenewal(t *testing.T) {
	req, err := http.NewRequest("PUT", localhostAndPort+"/licenses/08d7cf49-d3b2-4183-9971-66ceb1636f8e/renew?name=TESTDEVICE&id=9e29aa7c-9105-42ad-b344-c0c4bfbaa529&end=today", nil)
	//req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept-Language", "Ro_ro")
	req.Header.Set("Ignored", "Ignored header")
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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v", problem)
	}
}

func TestLendingCancellation(t *testing.T) {
	var buf bytes.Buffer
	payload := model.LicenseStatus{}
	enc := json.NewEncoder(&buf)
	enc.Encode(payload)

	req, err := http.NewRequest("PATCH", localhostAndPort+"/licenses/08d7cf49-d3b2-4183-9971-66ceb1636f8e/status?name=TESTDEVICE&id=9e29aa7c-9105-42ad-b344-c0c4bfbaa529&end=today", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("badu:hello")))
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept-Language", "Ro_ro")
	req.Header.Set("Ignored", "Ignored header")
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

	if resp.StatusCode < 300 {
		for hdrKey := range resp.Header {
			t.Logf("Header : %s = %s", hdrKey, resp.Header.Get(hdrKey))
		}
		t.Logf("response : %#v", string(body))
	} else {
		var problem http.Problem
		err = json.Unmarshal(body, &problem)
		if err != nil {
			t.Fatalf("Error Unmarshaling problem : %v.\nServer response : %s", err, string(body))
		}
		t.Logf("error response : %#v (%s)", problem, string(body))
	}
}
