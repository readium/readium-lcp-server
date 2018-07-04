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

package model_test

import (
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"os"
	"testing"
)

var stor model.Store

// run tests for frontend, lcp and lsd by changing this var
var whichServer = "lsd"
var sqlServer = "sqlite" //"mysql"

func TestMain(m *testing.M) {
	var err error
	// Start logger
	log := logger.New()
	info := ""
	switch sqlServer {
	case "sqlite":
		info = "sqlite3://file:/D:/GoProjects/src/readium-lcp-server/" + whichServer + ".sqlite?cache=shared&mode=rwc"
	case "mysql":
		info = "mysql://root:@tcp(127.0.0.1:3306)/readium?charset=utf8&parseTime=True&loc=Local"
	case "postgres":
		// TODO : add postgres init
		info = ""
	}

	// Prepare database
	stor, err = model.SetupDB(info, log, true)
	if err != nil {
		panic("Error setting up the database : " + err.Error())
	}
	switch whichServer {
	default:
		// front end migration
		err = stor.AutomigrateForFrontend()
		if err != nil {
			panic("Error migrating frontend database : " + err.Error())
		}
	case "lcp":
		err = stor.AutomigrateForLCP()
		if err != nil {
			panic("Error migrating LCP database : " + err.Error())
		}
	case "lsd":
		err = stor.AutomigrateForLSD()
		if err != nil {
			panic("Error migrating LSD database : " + err.Error())
		}
	}
	v := m.Run()
	os.Exit(v)
}
