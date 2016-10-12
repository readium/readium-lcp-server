package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/license_statuses"
	"github.com/readium/readium-lcp-server/localization"
	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/lsdserver/server"
	"github.com/readium/readium-lcp-server/transactions"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var config_file, host, publicBaseUrl, dbURI string
	var readonly bool = false
	var port int
	var err error

	if config_file = os.Getenv("READIUM_LICENSE_CONFIG"); config_file == "" {
		config_file = "config.yaml"
	}

	config.ReadConfig(config_file)

	err = localization.InitTranslations()
	if err != nil {
		panic(err)
	}

	readonly = config.Config.LsdServer.ReadOnly

	if host = config.Config.LsdServer.Host; host == "" {
		host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}
	if port = config.Config.LsdServer.Port; port == 0 {
		port = 8990
	}
	if publicBaseUrl = config.Config.LsdServer.PublicBaseUrl; publicBaseUrl == "" {
		publicBaseUrl = "http://" + host + ":" + strconv.Itoa(port)
		config.Config.LsdServer.PublicBaseUrl = publicBaseUrl
	}

	if dbURI = config.Config.LsdServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:test.sqlite?cache=shared&mode=rwc"
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

	hist, err := licensestatuses.Open(db)
	if err != nil {
		panic(err)
	}

	trns, err := transactions.Open(db)
	if err != nil {
		panic(err)
	}

	authFile := config.Config.LsdServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}

	htpasswd := auth.HtpasswdFileProvider(authFile)
	authenticator := auth.NewBasicAuthenticator("Basic Realm", htpasswd)

	complianceMode := config.Config.Logging.ComplianceTestsModeOn
	logDirectory := config.Config.Logging.LogDirectory
	err = logging.Init(logDirectory, complianceMode)
	if err != nil {
		panic(err)
	}

	HandleSignals()
	s := lsdserver.New(":"+strconv.Itoa(port), readonly, &hist, &trns, authenticator)
	if readonly {
		log.Println("License status server running in readonly mode on port " + strconv.Itoa(port))
	} else {
		log.Println("License status server running on port " + strconv.Itoa(port))
	}
	log.Println("Using database " + dbURI)
	log.Println("Public base URL=" + publicBaseUrl)

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
