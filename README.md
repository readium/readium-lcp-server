lcpserve
========

A Golang server that implements the LCP spec


Install
======

Assuming a working Go installation,

go get github.com/readium/readium-lcp-server

Running
=======
You must have a certificate (currently, only RSA is supported). Use the following environment variables to configure the lcp server:
- CERT - Points to the certificate file (typically a .crt)
- PRIVATE_KEY - Points to the private key (typically a .pem)
- PORT - Where lcpserve will listen, by default 8989
- HOST - The public hostname, defaults to `hostname`
- READONLY - Readonly mode for demo purposes
- DB - the connection string to the database, by default sqlite3://file:lcpserve.sqlite?cache=shared&mode=rwc

$GOPATH/bin/readium-lcp-server

Contributing
============
A great tool to help development is goconvey (available at https://github.com/smartystreets/goconvey). Running it will give you an auto-refreshing view of the tests in a nice Web UI.
