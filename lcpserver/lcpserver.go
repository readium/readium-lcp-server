package lcpserver

import (
	"crypto/tls"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/lcpserver"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
)

func main() {
	var config_file, host, port, publicBaseUrl, dbURI, storagePath, certFile, privKeyFile, static string
	var readonly bool = false
	var err error

	if host = os.Getenv("HOST"); host == "" {
		host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}

	if config_file = os.Getenv("READIUM_LCP_CONFIG"); config_file == "" {
		config_file = "lcpconfig.yaml"
	}

	config.ReadConfig(config_file)

	readonly = os.Getenv("READONLY") != ""

	if port = os.Getenv("PORT"); port == "" {
		port = "8989"
	}

	publicBaseUrl = config.Config.PublicBaseUrl
	if publicBaseUrl == "" {
		publicBaseUrl = "http://" + host + ":" + port
	}

	dbURI = config.Config.Database
	if dbURI == "" {
		if dbURI = os.Getenv("DB"); dbURI == "" {
			dbURI = "sqlite3://file:test.sqlite?cache=shared&mode=rwc"
		}
	}

	storagePath = config.Config.Storage.FileSystem.Directory
	if storagePath == "" {
		if storagePath = os.Getenv("STORAGE"); storagePath == "" {
			storagePath = "files"
		}
	}

	certFile = config.Config.Certificate.Cert
	privKeyFile = config.Config.Certificate.PrivateKey

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

	license.CreateLinks()

	var store storage.Store

	if mode := config.Config.Storage.Mode; mode == "s3" {
		s3Conf := s3ConfigFromYAML()
		store, _ = storage.S3(s3Conf)
	} else {
		os.Mkdir(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		store = storage.NewFileSystem(storagePath, publicBaseUrl+"/files")
	}

	packager := pack.NewPackager(store, idx, 4)

	static = config.Config.Static.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "/static")
	}

	HandleSignals()

	s := server.New(":"+port, static, readonly, &idx, &store, &lst, &cert, packager)
	s.ListenAndServe()
}
