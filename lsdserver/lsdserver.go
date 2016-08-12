package lsdserver

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/history"
	"github.com/readium/readium-lcp-server/lsdserver/server"
	"github.com/readium/readium-lcp-server/transactions"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var config_file, host, port, publicBaseUrl, dbURI, static string
	var readonly bool = false
	var err error

	if host = os.Getenv("LSD_HOST"); host == "" {
		host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}

	if config_file = os.Getenv("READIUM_LSD_CONFIG"); config_file == "" {
		config_file = "lsdconfig.yaml"
	}

	config.ReadConfig(config_file)

	readonly = os.Getenv("READONLY") != ""

	if port = os.Getenv("LSD_PORT"); port == "" {
		port = "8990"
	}

	publicBaseUrl = config.Config.PublicBaseUrl
	if publicBaseUrl == "" {
		publicBaseUrl = "http://" + host + ":" + port
	}

	dbURI = config.Config.Database
	if dbURI == "" {
		if dbURI = os.Getenv("LSD_DB"); dbURI == "" {
			dbURI = "sqlite3://file:test.sqlite?cache=shared&mode=rwc"
		}
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

	hist, err := history.Open(db)
	if err != nil {
		panic(err)
	}

	trns, err := transactions.Open(db)
	if err != nil {
		panic(err)
	}

	static = config.Config.Static.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../static")
	}

	HandleSignals()

	s := lsdserver.New(":"+port, static, readonly, &hist, &trns)
	s.ListenAndServe()
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
