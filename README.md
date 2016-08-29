Readium LCP Server
==================

Requirements
============

No binaries are currently pre-built, so you need to get a working Golang installation. Please refer to the official documentation for
installation procedures at https://golang.org/.

In order to keep the content keys for each encrypted EPUB, the server requires a SQL Database. The server currently includes drivers
for SQLite (the default option, and should be fine for small to medium installations) as well as MySQL and Postgres.

If you wish to use the external licenses, where a client gets a simple json file that contains instructions on how to fetch the encrypted EPUBS,
a publicly accessible folder must be made available for the server to store the file.

You must obtain a X.509 certificate through the Readium Foundation in order for your licenses to be accepted by the Reading Systems.

Install
=======

Assuming a working Go installation,
```sh
go get github.com/readium/readium-lcp-server

go build github.com/readium/readium-lcp-server/lcpencrypt
go build github.com/readium/readium-lcp-server/lcpserver
go build github.com/readium/readium-lcp-server/lsdserver 
```

Usage
=====

*Please note that the LCP Server currently does not include any authentication. Make sure it is only available to your internal services or add an authenticating
proxy in front of it*

The server is controlled by a configuration file "config.yaml".  
This file normally resides in the same directory of the executable but can be changed using the environment variable READIUM_LICENSE_CONFIG. 

See the example file of the config.yaml.sample 

license:
         links:
                      hint: "http://example.com/hint" 
                      status: "http://example.com/licenses/{license_id}/status" 
					  publication: "http://example.com/contents/{publication_id}" 
localization: 
        languages: ["ru-RU", "en-US"]
        folder: E:\GoProjects\src\github.com\readium\readium-lcp-server\messages\
        default_language: "en-US"
static:
    directory: "../static"
                 
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

# [lcpencrypt] 

encrypts epub file 

# [lcpserver]

add content, creates licenses, list licenses

# [lsdserver]

status document server

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server


[lcpencrypt]:<https://github.com/readium/readium-lcp-server/wiki>
[lcpserver]:<https://github.com/readium/readium-lcp-server/wiki>
[lsdserver]:<https://github.com/readium/readium-lcp-server/wiki>
