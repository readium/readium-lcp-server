package model_test

import (
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/model"
	"os"
	"testing"
)

var stor model.Store

// run tests for frontend, lcp and lsd by changing this var
var whichServer = "lut"
var sqlServer = "mysql" //"mysql"

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
