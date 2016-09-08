Readium LCP Server
==================

Requirements
============

No binaries are currently pre-built, so you need to get a working Golang installation. Please refer to the official documentation for
installation procedures at https://golang.org/.

In order to keep the content keys for each encrypted EPUB, the server requires an SQL Database. The server currently includes drivers
for SQLite (the default option, which should be fine for small to medium installations) as well as MySQL and Postgres.

If you wish to use the external licenses, where a client gets a simple json file that contains instructions on how to fetch the encrypted EPUB file,
a publicly accessible folder must be made available for the server to store the file.

You must obtain a X.509 certificate through EDRLab in order for your licenses to be accepted by Readium LCP compliant Reading Systems.

Install
=======

Assuming a working Go installation, the following will install the three executables that constitute a complete Readium LCP Server.

If you want to use the master branch:
```sh
// from the go workspace
cd $GOPATH
// get the different packages and their dependencies, then installs the packages
go get github.com/readium/readium-lcp-server
```

If you want to use a feature/F branch:
```sh
// from the go workspace
cd $GOPATH
// create the project repository
mkdir -p src/github.com/readium/readium-lcp-server
// clone the repo, selecting the development branch
git clone -b feature/F https://github.com/readium/readium-lcp-server.git src/github.com/readium/readium-lcp-server
// move to the project repository
cd src/github.com/readium/readium-lcp-server
// get the different packages and their dependencies, then installs the packages (dot / triple dot pattern)
go get ./...
```

You may prefer to install only some of the three executables. 
In such a case, the "go get" command should be called once for each package, e.g. for the lcpserver from the master branch:
```sh
// from the go workspace
cd $GOPATH
// get the different packages and their dependencies, then installs the packages
go get github.com/readium/readium-lcp-server/lcpserver
```

Usage
=====

The server is controlled by a yaml configuration file (e.g. "config.yaml").  
This file normally resides in the same directory of the executable but the path to this configuration file can be changed using the environment variable READIUM_LICENSE_CONFIG. 

An example config.yaml file exists with the name config.yaml.sample.

"certificate:"				
- cert: Points to the certificate file (a .crt)
- private_key: Points to the private key (a .pem)

"lcp:" & "lsd:" have a similar server structure 
- server:port: where lcpserve will listen, by default 8989
- server: host: the public hostname, defaults to `hostname`
- server:readonly: [true|false] readonly mode for demo purposes, no new file can be packaged
- server:database: the connection string to the database, by default sqlite3://file:lcpserve.sqlite?cache=shared&mode=rwc

static: points to the path where the /manage/index.html can be found

"license"
 - links:
	"hint" and "publication" default values.  If this value does not exist in the partial license, it will be added using this value.  If no value is present in the configuration file and no value is given to the partial license passed to the server, the server will reply with a 500 Server Error when asking to create a license.
- status : if present a lsdserver will be used to verify the license


The server includes a basic web interface that can be reached at http\://HOST:PORT/manage/. You can drag and drop EPUB files to encrypt them,
as well as emit licenses for the currently encrypted EPUBs.

Executables
===========

# [lcpencrypt]   (todo : wiki ? reference to google doc ? )

Allows to encrypt an epub file (on a different server) that can be added to the lcpserver 

# [lcpserver]

* encrypts & adds content, 
* creates licenses, 
* lists content 
* lists licenses

# [lsdserver]

status document server

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server


[lcpencrypt]:<https://github.com/readium/readium-lcp-server/wiki>
[lcpserver]:<https://github.com/readium/readium-lcp-server/wiki>
[lsdserver]:<https://github.com/readium/readium-lcp-server/wiki>
