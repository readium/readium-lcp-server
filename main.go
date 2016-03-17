package main

import (
	"crypto/tls"
	"database/sql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/go-sql-driver/mysql"
	"path/filepath"
	"runtime"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/server"
	"github.com/readium/readium-lcp-server/storage"
	"os"
	"strings"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var host, port, dbURI, storagePath, certFile, privKeyFile, static string
	var readonly bool = false
	var err error

	if host = os.Getenv("HOST"); host == "" {
		host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}

	config_file := "config.yaml"

	config, err := yaml.ReadFile(config_file)
	if err != nil {
		panic("can't read config file : " + config_file)
	}

	readonly = os.Getenv("READONLY") != ""

	if port = os.Getenv("PORT"); port == "" {
		port = "8989"
	}

	dbURI, _ = config.Get("database")
	if dbURI == "" {
		if dbURI = os.Getenv("DB"); dbURI == "" {
			dbURI = "sqlite3://file:test.sqlite?cache=shared&mode=rwc"
		}
	}

	storagePath, _ = config.Get("storage.filesystem.storage")
	if storagePath == "" {
		if storagePath = os.Getenv("STORAGE"); storagePath == "" {
			storagePath = "files"
		}
	}

	certFile, _ = config.Get("certificate.cert")
	privKeyFile, _ = config.Get("certificate.private_key")

	if certFile == "" {
		if certFile = os.Getenv("CERT"); certFile == "" {
			panic("Must specify a certificate")
		}
	}

	if privKeyFile == "" {
		if privKeyFile = os.Getenv("PRIVATE_KEY"); privKeyFile == "" {
			panic("Must specify a private key")
		}
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

	os.Mkdir(storagePath, os.ModePerm) //ignore the error, the folder can already exist
	store := storage.NewFileSystem(storagePath, "http://"+host+":"+port+"/files")
	if err != nil {
		panic(err)
	}


	static, _ = config.Get("static.directory");
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "/static")
	}

	s := server.New(":"+port, static, readonly, &idx, &store, &lst, &cert)
	s.ListenAndServe()
}
