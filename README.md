Readium LCP Server
==================

Documentation
============
Detailed documentation can be found in the [Wiki pages](../../wiki) of the project.

Prerequisites
=============

No binaries are currently pre-built, so you need to get a working Golang installation. Please refer to the official documentation for
installation procedures at https://golang.org/.

The servers require the setup of an SQL Database. A SQLite db is used by default (it should be fine for small to medium installations), and if the "database" property of each server defines a sqlite3 driver, the db setup is dynamically achieved when the server runs for the first time. 

A MySQL db creation script is provided as well, in the "dbmodel" folder; we expect other drivers (PostgresQL ...) to be provided by the community. Such script should be run be launching the servers.

If you wish to use external licenses, where a client gets a simple json file that contains instructions on how to fetch the encrypted EPUB file,
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


## [frontend]

A Frontend Test server, which offers a GUI and API for having a complete experience.

Public functionalities (accessible from the web):
* Fetch a license from its id
* Fetch a licensed publication from the license id


Install
=======

Assuming a working Go installation, the following will install the three executables that constitute a complete Readium LCP Server.

If you want to use the master branch:
```sh
// from the go workspace
cd $GOPATH
```

Alternatively, if you want to use a feature/F branch:
```sh
// from the go workspace
cd $GOPATH
// create the project repository
mkdir -p src/github.com/readium/readium-lcp-server
// clone the repo, selecting the development branch
git clone -b feature/F https://github.com/readium/readium-lcp-server.git src/github.com/readium/readium-lcp-server
```
Then fetch, build and install the different modules with:
```sh
// move to the project repository
cd src/github.com/readium/readium-lcp-server
// get the different packages and their dependencies, then installs the packages (dot / triple dot pattern)
go get ./...
````

You may prefer to install only some of the three executables. 
In such a case, the "go get" command should be called once for each package, e.g. for the lcpserver from the master branch:
```sh
cd $GOPATH
go get github.com/readium/readium-lcp-server/lcpserver
```

To install properly the Frontend Test Server, you must also install several npm packages.

Move to $GOPATH/src/github.com/readium/readium-lcp-server/frontend/manage
To install the packages and test your install, type
```sh
npm install
npm start
````

If this gives no error, your install is ready; type Ctrl-C to move out of the test mode. In case of errors, read Readme.md in the "manage" directory to get more details.

Configuration
==============

The server is controlled by a yaml configuration file (e.g. "config.yaml").  

The License Server, License Status Server and Frontend test server will search their configuration file in the bin directory by default; but the path to this file can be changed using the environment variable:

* READIUM_LCPSERVER_CONFIG for the LCP server
* READIUM_LSDSERVER_CONFIG for the LSD server
* READIUM_FRONTEND_CONFIG for the frontend test server

The value of the three global variables will be on the form /<path>/lcpconfig.yaml.

The three servers may share the same configuration file (if they are both executed on the same server) or they may have their own configuration file. 

The LCP and LSD servers also require authenticated API requests for some of their functionalities. A password file name .htpasswd must therefore be created to handle such authentication data, for each module. Like the configuration file, the .htpasswd file may be shared between the two modules.

A source example for creating a password file is http://www.htaccesstools.com/htpasswd-generator/. 
The htpasswd file format is e.g.:
```sh
	User1:$apr1$OMWGq53X$Qf17b.ezwEM947Vrr/oTh0
	User2:$apr1$lldfYQA5$8fVeTVyKsiPeqcBWrjBKMT
```

Here are details about the configuration properties:

*License Server*

"profile": value of the LCP profile; values are:
- "basic" (default value, as described in the Readium LCP specification, used for tests only);
- "1.0" (i.e. the current production profile, managed by EDRLab).

"lcp" section: parameters associated with the License Server.
- "host": the public server hostname, `hostname` by default
- "port": the listening port, `8989` by default
- "public_base_url": the public base URL, combination of the host and port values on http by default 
- "database": the URI formatted connection string to the database, `sqlite3://file:lcp.sqlite?cache=shared&mode=rwc` by default
- "auth_file": mandatory; the authentication file (an .htpasswd). Passwords must be encrypted using MD5.

"storage" section: parameters related to the storage of encrypted publications. 
- "filesystem" section: parameters related to a file system storage.
  - "directory": absolute path to the directory in which the encrypted publications are stored.

"certificate" section:	parameters related to the signature of licenses: 	
- "cert": the provider certificate file (.pem or .crt). It will be inserted in the licenses and used by clients for checking the signature.
- "private_key": the private key (.pem). It will be used for signing  licenses.

"license" section: parameters related to static information to be included in all licenses generated by the License Server:
- "links": links that will be included in all licenses. "hint" and "publication" links are required in a Readium LCP license.
  If no such link exists in the partial license passed from the frontend when a new license his requested, 
  these link values will be inserted in the partial license.  
  If no value is present in the configuration file and no value is inserted in the partial license, 
  the License server will reply with a 500 Server Error at license creation.
  The sub-properties of the "links" section are:
  - "hint": required; location where a Reading System can redirect a User looking for additional information about the User Passphrase. 
  - "publication": optional, templated URL; 
    location where the Publication associated with the License Document can be downloaded.
    The publication identifier is inserted via the variable {publication_id}.
  - "status": optional, templated URL; location of the Status Document associated with a License Document.
    The license identifier is inserted via the variable {license_id}.

"lsd_notify_auth" section: authentication parameters used by the License Server for notifying the License Status Server 
of a license generation. The notification endpoint is configured in the "lsd" section.
- "username": mandatory, authentication username
- "password": mandatory, authentication password

Here is a License Server sample config (assuming the License Status Server is using the 'basic' LCP profile, is active on http://127.0.0.1:8990 and the Frontend Server is active on http://127.0.0.1:8991):
```json
profile: "basic"
lcp:
    host: "127.0.0.1"
    port: 8989
    public_base_url: "http://127.0.0.1:8989"
    database: "sqlite3://file:/readiumlcp/lcpdb/lcp.sqlite?cache=shared&mode=rwc"
    auth_file: "/readiumlcp/lcpconfig/htpasswd"
storage:
    filesystem:
        directory: "/readiumlcp/lcpfiles/storage"
certificate:
    cert: "/readiumlcp/lcpconfig/cert.pem"
    private_key: "/readiumlcp/lcpconfig/privkey.pem"
license:
    links:
        status: "http://127.0.0.1:8990/licenses/{license_id}/status"     
        hint: "http://127.0.0.1:8991/static/hint.html"
        publication: "http://127.0.0.1:8991/licenses/{license_id}/publication" 
lsd:
    public_base_url:  "http://127.0.0.1:8990"
lsd_notify_auth: 
    username: "adm_username"
    password: "adm_password"

```

*License Status Server*

"lsd" section: parameters associated with the License Status Server. 
- "host": the public server hostname, `hostname` by default
- "port": the listening port, `8990` by default
- "public_base_url": the public base URL, combination of the host and port values on http by default 
- "database": the URI formatted connection string to the database, `sqlite3://file:lsd.sqlite?cache=shared&mode=rwc` by default
- "auth_file": mandatory; the authentication file (an .htpasswd). Passwords must be encrypted using MD5.

- "license_link_url": the url template representing the url from which a license can be fetched from the provider's frontend. This url will be inserted in the 'license' link of every status document.

"license_status" section: parameters related to the interactions implemented by the License Status server, if any:
- "renting_days": default number of days allowed for a loan. 
- "renew": boolean; if `true`, the renewal of a loan is possible. 
- "renew_days": default number of additional days allowed after a renewal.
- "return": boolean; if `true`,  an early return is possible.  
- "register": boolean; if `true`,  registering a device is possible.

"logging" section: parameters for logging results of API method calls on the License Status server:
- "log_directory": the complete path to the log file.
- "compliance_tests_mode_on": boolean; if `true`, logging is turned on.

"lcp_update_auth" section: authentication parameters used by the License Status Server for updating a license via the License Server. The notification endpoint is configured in the "lcp" section.
- "username": mandatory, authentication username
- "password": mandatory, authentication password

Here is a License Status Server sample config (assuming the License Status Server is active on http://127.0.0.1:8990 and the Frontend Server is active on http://127.0.0.1:8991):
```json
lsd:
    host: "127.0.0.1"
    port: 8990
    public_base_url: "http://127.0.0.1:8990"
    database: "sqlite3://file:/readiumlcp/lcpdb/lsd.sqlite?cache=shared&mode=rwc"
    auth_file: "/Users/laurentlemeur/Work/lcpconfig/htpasswd"
    license_link_url: "http://127.0.0.1:8991/licenses/{license_id}"
license_status:
    register: true
    renew: true
    return: true
    renting_days: 60
    renew_days: 7
logging: 
    log_directory: "/readiumlcp/lcpfiles/lsdserver.log"
    compliance_tests_mode_on: false
lcp:
  public_base_url:  "http://127.0.0.1:8989"
lcp_update_auth: 
    username: "adm_username"
    password: "adm_password"
```

*Frontend Server*

"frontend" section: parameters associated with the Frontend Test Server. 
- "host": the public server hostname, `hostname` by default
- "port": the listening port, `8991` by default
- "public_base_url": the public base URL, combination of the host and port values on http by default 
- "database": the URI formatted connection string to the database, `sqlite3://file:frontend.sqlite?cache=shared&mode=rwc` by default
- "master_repository": repository where the uploaded EPUB files are stored before encryption. 
- "encrypted_repository": repository where the encrypted EPUB files are stored after upload.
- "directory": the directory containing the client app; by default $GOPATH/src/github.com/readium/readium-lcp-server/frontend/manage.
- "provider_uri": provider uri, which will be inserted in all licenses produced via this test frontend.
- "right_print": allowed number of printed pages, which will be inserted in all licenses produced via this test frontend.
- "right_copy": allowed number of copied characters, which will be inserted in all licenses produced via this test frontend.

The config file of a Frontend Test Server must define a "lcp" "public_base_url", "lsd" "public_base_url", "lcp_update_auth" "username" and "password", and "lsd_notify_auth" "username" and "password".

Here is a Frontend Test Server sample config:
```json
frontend:
    host: "127.0.0.1"
    port: 8991
    database: "sqlite3://file:/readiumlcp/lcpdb/frontend.sqlite?cache=shared&mode=rwc"
    master_repository: "/readiumlcp/lcpfiles/master"
    encrypted_repository: "/readiumlcp/lcpfiles/encrypted"
    directory: "/src/github.com/readium/readium-lcp-server/frontend/manage"
    provider_uri: "https://www.edrlab.org"
    right_print: 10
    right_copy: 2000
lcp:
  public_base_url:  "http://127.0.0.1:8989"
lsd:
  public_base_url:  "http://127.0.0.1:8990"
lcp_update_auth: 
    username: "adm_username"
    password: "adm_password"
lsd_notify_auth: 
    username: "adm_username"
    password: "adm_password"

```

*And for all servers*

"localization" section: parameters related to the localization of the messages sent by all three servers.
- "languages": array of supported localization languages
- "folder": point to localization file (a .json)
- "default_language": default language for localization

NOTE: the localization file names (ex: 'en-US.json, de-DE.json') must match the set of supported localization languages.

NOTE: a CBC / GCM configurable property has been DISABLED, see https://github.com/readium/readium-lcp-server/issues/109
"aes256_cbc_or_gcm": either "GCM" or "CBC" (which is the default value). This is used only for encrypting publication resources, not the content key, not the user key check, not the LCP license fields.

Execution
==========
each server must be launched in a different context (i.e. a different shell for local use), from 
 $GOPATH/src/github.com/readium/readium-lcp-server

Each server is executed with no parameter:
- lcpserver
- lsdserver
- frontend

After the frontend server is launched, you can access to the server GUI via its base url, e.g. http://http://127.0.0.1:8991

NOTE: even if you deploy the server locally, using 127.0.0.1 is not recommended, as you won't be able to access the modules from e.g. a mobile app. It's much better to use the WiFi IPv4 address to your computer, and access the server from your mobile device via WiFi.  

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server
