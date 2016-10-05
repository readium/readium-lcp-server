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

Executables
===========
The server software is composed of three independant parts:

## [lcpencrypt]  

A command line utility for EPUB content encryption. This utility can be included in any processing pipeline. 

* takes one unprotected EPUB 3 file as input and generates an encrypted file as output.
* notifies the License server of the generation of an encrypted file.

## [lcpserver]

A License server, which implements Readium Licensed Content Protection 1.0.

Private functionalities (authentication needed):
* Store the data resulting from an external encryption
* Generate a license
* Generate a protected publication
* Update the rights associated with a license
* Get a set of licenses
* Get a license


## [lsdserver]

A License Status server, which implements Readium License Status Document 1.0.

Public functionalities (accessible from the web):
* Return a license status document
* Process a device registration
* Process a lending return
* Process a lending renewal

Private functionalities (authentication needed):
* Create a license status document
* Filter licenses
* List all registered devices for a given licence
* Revoke/cancel a license


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

Server Configuration
====================

The server is controlled by a yaml configuration file (e.g. "config.yaml").  
This file normally resides in the bin directory but the path to this configuration file can be changed using the environment variable READIUM_LICENSE_CONFIG.

"certificate":	parameters related to the signature of the licenses	
- "cert": the provider certificate file (.pem or .crt). It will be inserted in the licenses and used by clients for checking the signature.
- "private_key": the private key (.pem). It will be used for signing  licenses.

"lcp" (License Server) & "lsd" (License Status Server) have an identical structure:
- "host": the public server hostname, `hostname` by default
- "port": the listening port, `8989` by default
- "public_base_url": the public base URL, combination of the host and port values on http by default 
- "database": the URI formatted connection string to the database, `sqlite3://file:lcpserve.sqlite?cache=shared&mode=rwc` by default
- "auth_file": mandatory; the authentication file (an .htpasswd). Passwords must be encrypted using MD5.
	The source example for creating password is http://www.htaccesstools.com/htpasswd-generator/. 
	The format of the file is:
```sh
	User1:$apr1$OMWGq53X$Qf17b.ezwEM947Vrr/oTh0
	User2:$apr1$lldfYQA5$8fVeTVyKsiPeqcBWrjBKMT
```

"storage": parameters related to the storage of the protected publications.
- "filesystem": parameters related to a file system storage
  - "directory": absolute path to the directory in which the protected publications are stored.

"license": parameters related to static information to be included in all licenses
- "links": links that will be included in all licenses. "hint" and "publication" links are required in a Readium LCP license.
  If no such link exists in the partial license passed from the frontend when a new license his requested, 
  these link values will be inserted in the partial license.  
  If no value is present in the configuration file and no value is inserted in the partial license, 
  the License server will reply with a 500 Server Error at license creation.
  - "hint": required; location where a Reading System can redirect a User looking for additional information about the User Passphrase. 
  - "publication": optional, templated URL; 
    location where the Publication associated with the License Document can be downloaded.
    The publication identifier is inserted via the variable {publication_id}.
  - "status" : optional, templated URL; location of the Status Document associated with a License Document.
    The license identifier is inserted via the variable {license_id}.

NOTE: here is a license section snippet:
```json
license:
    links:
        hint: "http://www.edrla.org/readiumlcp/hint.html"
		    publication: "http://www.edrla.org/readiumlcp/{publication_id}/publication" 
        status: "http://www.edrla.org/readiumlcp/{license_id}/status" 
```

"license_status": parameters related to the interactions implemented by the License Status server, if any
- renting_days: number of days be the license ends.
- renew: boolean; if `true`, rental renewal is possible. 
- renew_days: number of days added to the license if renewal is active.
- return: boolean; if `true`,  early return is possible.  
- register: boolean; if `true`,  registering a device is possible.

"localization": parameters related to the localization of the messages sent by the server
- languages: array of supported localization languages
- folder: point to localization file (a .json)
- default_language: default language for localization

NOTE: list files for localization (ex: 'en-US.json, de-DE.json') must match the array of supported localization languages

"logging": parameters for logging results of API methods
- log_directory: point to log file (a .log).
- compliance_tests_mode_on: boolean; if `true`, logging is turned on.

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server
