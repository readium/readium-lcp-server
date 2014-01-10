lcpserve
========

A Golang server that implements the LCP spec


Install
======

Assuming a working Go installation,

go get github.com/jpbougie/lcpserve

Running
=======
You must have a certificate (currently, only RSA is supported). Use the following environment variables to configure lcpserve:
- CER
- PRIVATE_KEY
- PORT
- HOST
- READONLY

$GOPATH/bin/lcpserve
