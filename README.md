Readium LCP Server
========

This server allows you to both encrypt EPUBs as well as deliver licenses in accordance with the Readium LCP specification.

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
======

Assuming a working Go installation,

go get github.com/readium/readium-lcp-server

Usage
=====

*Please note that the LCP Server currently does not include any authentication. Make sure it is only available to your internal services or add an authenticating
proxy in front of it*

The server is controlled by a set of environment variables. Here are their descriptions and possible values:

- CERT - Points to the certificate file (a .crt)
- PRIVATE_KEY - Points to the private key (a .pem)
- PORT - Where lcpserve will listen, by default 8989
- HOST - The public hostname, defaults to `hostname`
- READONLY - Readonly mode for demo purposes, no new file can be packaged
- DB - the connection string to the database, by default sqlite3://file:lcpserve.sqlite?cache=shared&mode=rwc

You can also use a YAML config file named config.yaml, which follows the structure presented in config.yaml.sample


Once those are set, you can run the server by calling the following:
$GOPATH/bin/readium-lcp-server


The server includes a basic web interface that can be reached at http://HOST:PORT/manage/. You can drag and drop EPUB files to encrypt them,
as well as emit licenses for the currently encrypted EPUBs.

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server
