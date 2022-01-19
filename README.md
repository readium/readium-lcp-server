Readium LCP Server
==================

Documentation
============
As a retailer, public library or specialized e-distributor, you are distributing EPUB and PDF files and you want them protected against oversharing by the Readium LCP DRM. Your distribution platform already handles publications, users and the purchases / loans and you can develop a REST interface between this platform and a License server. If you are in this situation, the open-source codebase is what you need. 

Using the Readium LCP Server you can:
* Encrypt your entire catalog of EPUB and PDF files and make these files ready for download from any LCP compliant user agents;
* Generate LCP licenses on the fly from your distribution platform, which will deliver them to the proper users and user agents;
* Let users request a loan extension or make an early loan return;
* Cancel a license in case a user has declared he wasn't able to user it (usually because he does not use an LCP compliant reading application);
* Revoke a license in case a user has overshared it. 
 
Detailed documentation can be found in the [Wiki pages](../../wiki) of the project.

Prerequisites
=============

No binaries are currently pre-built, so you need to get a working Golang installation. 
Please refer to the official documentation for installation procedures at https://golang.org/.

Install *go 1.13* or higher.

The servers require the setup of an SQL Database. 

- SQLite is sufficient for most needs. If the "database" property of each server defines a sqlite3 driver, the db setup is dynamically achieved when the server runs for the first time. SQLite database creation scripts are provided in the "dbmodel" folder in case they are useful. 
-  MySQL database creation scripts are provided as well in the "dbmodel" folder. These scripts should be run before launching the servers for the first time. 

We expect other drivers (PostgresQL ...) to be provided by the community. Some developers have deployed MS SQL Server, but the corresponding scripts were not provided so far. 

A major revision of the software will feature an ORM (Object Realtional Mapper), but it is still unsufficiently tested to be moved to the master branch. 

Your platform must be able to handle:

1/ the license server, active in your intranet, not accessible from the Web, only accessible from you frontend server via its REST API. 

2/ the license status server, accessible from the Web via its REST API.

3/ a large storage volume for encrypted publications, accessible in read mode from the Web via HTTP URLs (publications are encrypted once, every license generated for such publication is pointing at the same encrypted file.

You must obtain a X.509 certificate and confidential crypto information through EDRLab in order for your licenses to be accepted by Readium LCP compliant Reading Systems.

A folder publicly accessible from the Web must be made available for the server to store encrypted files.

Quickstart
==========


> You have to download node [v6.9.2](https://nodejs.org/dist/v6.9.2/) to compile the frontend webapp

```
make clean && PATH=/Users/edrlab/Downloads/node-v6.9.2-darwin-x64/bin:$PATH make && make run
```

## docker-compose

```
docker-compose -f docker/docker-compose.yml up
```

Executables
===========
The server software is composed of three independant parts:

## [lcpencrypt]  

A command line utility for content encryption. This utility can be included in any processing pipeline. 

lcpencrypt:
* Takes an unprotected publication as input and generates an encrypted file as output.
* Notifies the License server of the generation of the encrypted file.

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

A Test Frontend server, which mimics your own frontend platform (e.g. bookselling website), with a GUI and its own REST API. Its sole goal is to help you test the License and License status servers. 

Public functionalities (accessible from the web):
* Fetch a license from its id
* Fetch a licensed publication from the license id


Install
=======

Assuming a working Go installation and a properly set $GOPATH, the following will install the four executables that constitute a complete Readium LCP Server.

On Linux and MacOS:

If you want to use the master branch:
```sh
# from the go workspace
cd $GOPATH
# fetch, build and install the different packages and their dependencies
go get -v github.com/readium/readium-lcp-server/...
```

Warning: Go has a funny 3-dots syntax, and you really have to type "/..." at the end of the line. 

You should now find the generated Go binaries in $GOPATH/bin: 

- `lcpencrypt`: the command line encryption tool,
- `lcpserver`: the license server,
- `lsdserver`: the status document server,
- `frontend`: a test application (Test Frontend Server) which mimics your content management system. 

Alternatively, if you want to use a feature branch:
```sh
# from the go workspace
cd $GOPATH
# clone the repo, selecting the feature/F branch
git clone -b <feature-branch> https://github.com/readium/readium-lcp-server.git src/github.com/readium/readium-lcp-server
# then fetch, build and install the different packages and their dependencies
go get -v github.com/readium/readium-lcp-server/...
````

You may prefer to install only some of the executables. 
In such a case, the "go get" command should be called once for each package, e.g. for the lcpserver from the master branch:
```sh
cd $GOPATH
go get -v github.com/readium/readium-lcp-server/lcpserver
```

To install properly the Test Frontend Server, you must also install several npm packages.

Note: use a modern version of npm, e.g. npm 6.12.0 or higher. 

To install the npm packages, type:
```sh
cd $GOPATH/src/github.com/readium/readium-lcp-server/frontend/manage
npm install
````

On Windows 10:

You must first install a GCC compiler in order to compile the SQLite module and to move later to "production mode". [TDM-GCC](http://tdm-gcc.tdragon.net/download) gives great results. 

Also, in the previous instructions, replace: 

* $GOPATH with %GOPATH%
* forward slashes with backslashes in paths.

Configuration
==============

## Environment variables

The server is controlled by a yaml configuration file (e.g. "config.yaml") stored in a convenient folder, eg. `/usr/local/var/lcp`.  

The License Server, License Status Server and Frontend test server will search their configuration file in the go bin directory by default; but the path to the file should be changed using an environment variable:

* `READIUM_LCPSERVER_CONFIG` for the License server
* `READIUM_LSDSERVER_CONFIG` for the License Status server
* `READIUM_FRONTEND_CONFIG` for the Frontend test server

The value of the each global variable is an absolute path to the configuration file for the given server. The three servers may share the same configuration file (useful if they are executed on the same server) or each server may get its own configuration file; this is your choice. 

## Password file

The LCP and LSD servers also require authenticated API requests for some of their functionalities. A password file formatted as an Apache "htpasswd" file is used for such authentication data. The htpasswd file format is of the form:

```sh
	User:$apr1$OMWGq53X$Qf17b.ezwEM947Vrr/oTh0
```

An example of password file generator is found [here](http://www.htaccesstools.com/htpasswd-generator/). 

The password file may be shared between the LCP and LSD servers if the same credentials are used for both. The exact location and name of the file have no importance, as it will be referenced from the configuration file; but we recommand to name it `htpasswd` and place this file in the same folder as the configuration file, eg. `/usr/local/var/lcp`.

## Certificate

The LCP server requires an X509 certificate and its associated private key. The exact location and name of these files have no importance, as they will be referenced from the configuration file; but we recommand to keep the file name of the file provided by EDRLAb and place these files in a subfolder of the previous one, eg. `/usr/local/var/lcp/cert`.

A test certificate (`cert-edrlab-test.pem`) and private key (`privkey-edrlab-test.pem`) are provided in the `test/cert` directory of the project. These files are be used as long as the LCP server is configured in test mode (`profile` = `basic`). They are replaced by a provider specific certificate and private key when the server is moved to its production mode. 

## Quick-start configuration

A quick-start configuration meant only for test purposes is available in `test/config.yaml`. This file includes a default configuration for the the LCP, LSD and frontend servers.

1. Create a LCP_HOME folder, eg. `/usr/local/var/lcp`
2. Copy the `test/config.yaml` file into LCP_HOME 
3. Replace any occurrence of `<LCP_HOME>` in config.yaml with the absolute path to the LCP_HOME folder
4. Setup the `READIUM_*_CONFIG` env variables, which must reference the configuration file
5. Generate a password file and place it into LCP_HOME
6. Create `cert`, `db`, `files/storage` folders in LCP_HOME
7. Copy the test certificate, test private key into the `cert` subfolder of LCP_HOME

## Individual server configurations

Here are the details about the configuration properties of each server. In the samples, replace `<LCP_HOME>` with the absolute path to the folder containing the configuration file, password file, encrypted files, database and certificates.

### License Server

`profile`: value of the LCP profile; allowed values are:
- `basic`: default value, as described in the Readium LCP specification, used for tests only
- `1.0`: the current production profile, created by EDRLab.

`lcp` section: parameters associated with the License Server.
- `host`: the public server hostname, `hostname` by default
- `port`: the listening port, `8989` by default
- `public_base_url`: the public base URL, used by the license status server and the frontend test server to communicate with this server; combination of the host and port values on http by default, which is sufficient as the license server should not be visible from the Web.  
- `database`: the URI formatted connection string to the database, `sqlite3://file:lcp.sqlite?cache=shared&mode=rwc` by default
- `auth_file`: mandatory; the path to the password file introduced above. 

`storage` section: parameters related to the storage of encrypted publications.
- `mode` : optional. Possible values are "local" (default value) and "s3".

If `mode` value is `s3`:
- `endpoint` (optional): name of the target S3 endpoint, if one is defined in the AWS S3 setup.
- `region` (required): name of the target AWS region.
- `bucket` (required): name of the target S3 bucket.

If the storage is an S3 bucket, client credentials default to a chain of credential providers, searched in environment variables and a shared credential file. See [Setting up an S3 Storage](https://github.com/readium/readium-lcp-server/wiki/Setting-up-an-S3-storage) for details. 

Alternatively (and this is not recommended!), credentials can be stored in clear in the configuration file:
- `access_id`: value of the AWS access key id.
- `secret`: value of the AWS secret access key.

If `mode` value is NOT `s3`:
- `filesystem` subsection: parameters related to a file system storage.   
  - `directory`: absolute path to the directory in which the encrypted publications are stored. In production, this directory must be accessible from the Web via the URL defined in `license/links/publication` (see below) 
  This storage must be accessible from the Web via a simple URL, specified via the `license/publication` parameter.

`certificate` section:	parameters related to the signature of licenses: 	
- `cert`: the path to provider certificate file (.pem or .crt). It will be inserted in the licenses and used by clients for checking the signature. 
- `private_key`: the path to the private key (.pem) asociated with the certificate. It will be used for signing licenses. 

`license` section: parameters related to static information to be included in all licenses generated by the License Server:
- `links`: subsection: links that will be included in all licenses. `hint` and `publication` links are required in a Readium LCP license.
  If no such link exists in the partial license passed from the frontend when a new license his requested, 
  these link values will be inserted in the partial license.  
  If no value is present in the configuration file and no value is inserted in the partial license, 
  the License server will reply with a 500 Server Error at license creation.
  The sub-properties of the `links` section are:
  - `hint`: required; location where a Reading System can redirect a User looking for additional information about the User Passphrase. 
  - `publication`: optional, templated URL; 
    location where the encrypted Publication associated with the License Document will be downloaded from the Web. 
    This access point corresponds to the directory where encrypted publications are stored by the License Server (see `storage/filesystem/directory`).
    To expose this storage directory on the Web, the provider may decide to install a reverse-proxy, use a Web drive, use a CDN etc. This is a deployment choice which has nothing to do with this open-source projet.  
    During initial tests (before the License Server is hidden from the Web), this URL may simply be the one described described [here](https://github.com/readium/readium-lcp-server/wiki/LCP-License-Server-API#fetch-an-encrypted-publication). 
    The publication (alias content) identifier is inserted in the URL via the variable {publication_id}.
    Note that this is working because the file name of the stored encrypted publications is simply their publication identifier. 
  - `status`: optional, templated URL; location of the Status Document associated with a License Document.
    The license identifier is inserted via the variable {license_id}.

`lsd_notify_auth` section: authentication parameters used by the License Server for notifying the License Status Server 
of a license generation. The notification endpoint is configured in the `lsd` section.
- `username`: mandatory, authentication username
- `password`: mandatory, authentication password

Here is a License Server sample config (assuming the License Status Server is using the 'basic' LCP profile, is active on http://127.0.0.1:8990 and the Frontend Server is active on http://127.0.0.1:8991):
```json
profile: "basic"
lcp:
    host: "127.0.0.1"
    port: 8989
    public_base_url: "http://127.0.0.1:8989"
    database: "sqlite3://file:<LCP_HOME>/db/lcp.sqlite?cache=shared&mode=rwc"
    auth_file: "<LCP_HOME>/htpasswd"
storage:
    filesystem:
        directory: "<LCP_HOME>/files/storage"
certificate:
    cert: "<LCP_HOME>/cert/cert.pem"
    private_key: "<LCP_HOME>/cert/privkey.pem"
license:
    links:
        status: "http://127.0.0.1:8990/licenses/{license_id}/status"     
        hint: "http://127.0.0.1:8991/static/hint.html"
        publication: "http://127.0.0.1:8989/contents/{publication_id}" 

lsd:
    public_base_url:  "http://127.0.0.1:8990"
lsd_notify_auth: 
    username: "adm_username"
    password: "adm_password"

```

### License Status Server

`lsd` section: parameters associated with the License Status Server. 
- `host`: the public server hostname, `hostname` by default
- `port`: the listening port, `8990` by default
- `public_base_url`: the public base URL, used by the license server and the frontend test server to communicate with this server; combination of the host and port values on http by default; as this server is exposed on the Web in production, a domain name should be present in the URL.
- `database`: the URI formatted connection string to the database, `sqlite3://file:lsd.sqlite?cache=shared&mode=rwc` by default
- `auth_file`: mandatory; the path to the password file introduced above. 

- `license_link_url`: mandatory; the url template representing the url from which a license can be fetched from the provider's frontend server. This url will be inserted in the 'license' link of every status document. It must be the url of a server acting as a proxy between the user request and the License Server. Such proxy is mandatory, as the License Server  does not possess user information needed to craft a license from its identifier. If the test frontend server is used as a proxy, the url must be of the form "http://<frontend-server-url>/api/v1/licenses/{license_id}" (note the /api/v1 section).

`license_status` section: parameters related to the interactions implemented by the License Status server, if any:
- `renting_days`: maximum number of days allowed for a loan, from the date the loan starts. If set to 0 or absent, no loan renewal is possible. 
- `renew`: boolean; if `true`, the renewal of a loan is possible. 
- `renew_days`: default number of additional days allowed during a renewal.
- `return`: boolean; if `true`, an early return is possible.  
- `register`: boolean; if `true`, registering a device is possible.
- `renew_page_url`: URL; if set, the renew feature is implemented as an HTML page, using this URL. This is mostly useful for testing client applications.

`lcp_update_auth` section: authentication parameters used by the License Status Server for updating a license via the License Server. The notification endpoint is configured in the `lcp` section.
- `username`: mandatory, authentication username
- `password`: mandatory, authentication password

Here is a License Status Server sample config (assuming the License Status Server is active on http://127.0.0.1:8990 and the Frontend Server is active on http://127.0.0.1:8991):

```json
lsd:
    host: "127.0.0.1"
    port: 8990
    public_base_url: "http://127.0.0.1:8990"
    database: "sqlite3://file:<LCP_HOME>/db/lsd.sqlite?cache=shared&mode=rwc"
    auth_file: "<LCP_HOME>/htpasswd"
    license_link_url: "http://127.0.0.1:8991/api/v1/licenses/{license_id}"
license_status:
    register: true
    renew: true
    return: true
    renting_days: 60
    renew_days: 7

lcp:
  public_base_url:  "http://127.0.0.1:8989"
lcp_update_auth: 
    username: "adm_username"
    password: "adm_password"
```

### Frontend Server

`frontend` section: parameters associated with the Test Frontend Server. 
- `host`: the public server hostname, `hostname` by default
- `port`: the listening port, `8991` by default
- `public_base_url`: the public base URL, used to access the frontend UI; combination of the host and port values on http by default 
- `database`: the URI formatted connection string to the database, `sqlite3://file:frontend.sqlite?cache=shared&mode=rwc` by default
- `master_repository`: repository where the uploaded EPUB files are stored before encryption. 
- `encrypted_repository`: repository where the encrypted EPUB files are stored after upload. The LCP server must have access to the path declared here, as it will move each encrypted file to its final storage folder on notification of encryption from the Frontend Server. 
- `directory`: the directory containing the client web app; by default $GOPATH/src/github.com/readium/readium-lcp-server/frontend/manage.
- `provider_uri`: provider uri, which will be inserted in all licenses produced via this test frontend.
- `right_print`: allowed number of printed pages, which will be inserted in all licenses produced via this test frontend.
- `right_copy`: allowed number of copied characters, which will be inserted in all licenses produced via this test frontend.

The config file of a Test Frontend Server must also define the following properties: 

`lcp`
- `public_base_url`

`lsd`
- `public_base_url`

`lcp_update_auth`
- `username`
- `password`

`lsd_notify_auth`
- `username`
- `password`

Here is a Test Frontend Server sample config:
```json
frontend:
    host: "127.0.0.1"
    port: 8991
    database: "sqlite3://file:<LCP_HOME>/db/frontend.sqlite?cache=shared&mode=rwc"
    master_repository: "<LCP_HOME>/files/master"
    encrypted_repository: "<LCP_HOME>/files/encrypted"
    provider_uri: "https://www.myprovidername.org"
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

### And for all servers

`localization` section: parameters related to the localization of the messages sent by all three servers.
- `languages`: array of supported localization languages
- `folder`: point to localization file (a .json)
- `default_language`: default language for localization

NOTE: the localization file names (ex: 'en-US.json, de-DE.json') must match the set of supported localization languages.

NOTE: a CBC / GCM configurable property has been DISABLED, see https://github.com/readium/readium-lcp-server/issues/109
"aes256_cbc_or_gcm": either "GCM" or "CBC" (which is the default value). This is used only for encrypting publication resources, not the content key, not the user key check, not the LCP license fields.

Execution
==========
Each server must be launched in a different context (i.e. a different terminal for local use). If the path to the generated Go binaries ($GOPATH/bin) is properly set, each server can launched from any location:

- `lcpserver`
- `lsdserver`
- `frontend`

After the frontend server has been launched, you can access the server GUI via its base url, e.g. http://127.0.0.1:8991

NOTE: even if you deploy the server locally, using 127.0.0.1 is not recommended, as you won't be able to access the modules from e.g. a mobile app. It is much better to use the WiFi IPv4 address of your computer and access the server from a mobile device via WiFi.  

Contributing
============
Please make a Pull Request with tests at github.com/readium/readium-lcp-server
