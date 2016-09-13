package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/abbot/go-http-auth"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/index"
	"github.com/readium/readium-lcp-server/lcpserver/server"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/pack"
	"github.com/readium/readium-lcp-server/storage"
)

func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	return parts[0], parts[1]
}

func main() {
	var config_file, host, publicBaseUrl, dbURI, storagePath, certFile, privKeyFile, static string
	var readonly bool = false
	var port int
	var err error

	if config_file = os.Getenv("READIUM_LICENSE_CONFIG"); config_file == "" {
		config_file = "config.yaml"
	}
	config.ReadConfig(config_file)
	log.Println("Reading config " + config_file)

	readonly = config.Config.LcpServer.ReadOnly
	if host = config.Config.LcpServer.Host; host == "" {
		host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}
	if port = config.Config.LcpServer.Port; port == 0 {
		port = 8989
	}
	if publicBaseUrl = config.Config.LcpServer.PublicBaseUrl; publicBaseUrl == "" {
		publicBaseUrl = "http://" + host + ":" + strconv.Itoa(port)
	}

	if dbURI = config.Config.LcpServer.Database; dbURI == "" {
		dbURI = "sqlite3://file:test.sqlite?cache=shared&mode=rwc"
	}
	if storagePath = config.Config.Storage.FileSystem.Directory; storagePath == "" {
		storagePath = "files"
	}
	if certFile = config.Config.Certificate.Cert; certFile == "" {
		panic("Must specify a certificate")
	}
	if privKeyFile = config.Config.Certificate.PrivateKey; privKeyFile == "" {
		panic("Must specify a private key")
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
		os.MkdirAll(storagePath, os.ModePerm) //ignore the error, the folder can already exist
		store = storage.NewFileSystem(storagePath, publicBaseUrl+"/files")
	}

	packager := pack.NewPackager(store, idx, 4)

	static = config.Config.Static.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../static")
	}

	authFile := config.Config.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}

	_, err = os.Stat(authFile)
	if err != nil {
		panic(err)
	}

	htpasswd := auth.HtpasswdFileProvider(authFile)
	authenticator := auth.NewBasicAuthenticator("Readium License Content Protection Server", htpasswd)

	HandleSignals()

	s := lcpserver.New(":"+strconv.Itoa(port), static, readonly, &idx, &store, &lst, &cert, packager, authenticator)
	if readonly {
		log.Println("License server running in readonly mode on port " + strconv.Itoa(port))
	} else {
		log.Println("License server running on port " + strconv.Itoa(port))
	}
	log.Println("using database " + dbURI)
	log.Println("Public base URL=" + publicBaseUrl)
	log.Println("License links:")
	for nameOfLink, link := range config.Config.License.Links {
		log.Println("  " + nameOfLink + " => " + link)
	}

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

func s3ConfigFromYAML() storage.S3Config {
	s3config := storage.S3Config{}

	s3config.Id = config.Config.Storage.AccessId
	s3config.Secret = config.Config.Storage.Secret
	s3config.Token = config.Config.Storage.Token

	s3config.Endpoint = config.Config.Storage.Endpoint
	s3config.Bucket = config.Config.Storage.Bucket
	s3config.Region = config.Config.Storage.Region

	s3config.DisableSSL = config.Config.Storage.DisableSSL
	s3config.ForcePathStyle = config.Config.Storage.PathStyle

	return s3config
}
