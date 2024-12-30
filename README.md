Readium LCP Server
==================

Documentation
============
As a retailer, public library or specialized e-distributor, you are distributing EPUB or PDF ebooks, audiobooks packaged using the LPF or RPF (.audiobook) formats, or comics packaged using the RPF (.divina) format. You want them protected by the Readium LCP DRM. Your CMS (Content Management system) already handles publications, users, purchases or loans, your technical team is able to integrate this CMS with a License server by creating a new http endpoint in the CMS and requesting the License Server via its REST interface, and this team is also able to compile and deploy a Go open-source codebase. If you are in this situation, this open-source codebase is made for you. 

Using the tools provided in this project you can:
* Encrypt your entire catalog of publications and store these encrypted files in a file system or S3 bucket, ready for download from any LCP compliant reading application;
* Generate LCP licenses;
* Let users request a loan extension or an early return;
* Cancel a license in case a user has declared he wasn't able to use it;
* Revoke a license in case of oversharing. 
 
**A detailed documentation is found in the [Wiki pages of the project](../../wiki). You really have to read it before you start testing this application.**

Prerequisites
=============

Binaries are only pre-built on demand and for a service fee, therefore in the general case you'll need to get a working Golang installation. 
Please refer to the official Go documentation for installation procedures at https://golang.org/.

This software is working with *go 1.19* or higher. It is currently maintained using *go 1.21* (February 2024). 

You must put in place:

1/ the License Server, active in your intranet, not accessible from the Web, only accessible from your CMS via a REST API. 

2/ the Status Server, accessible from the Web via a REST API, using https (you'll need to install a reverse proxy).

3/ a License Gateway, i.e. a piece of sofware you'll have to develop, which takes a request for an existing LCP license from a reading app, interrogates your database in order to get user information relative to this license, calls the License Server endpoint and returns this fresh LCP license back to the caller app (more information in the project Wiki).

4/ a large storage volume for encrypted publications. It can be either a file system accessible from the Web via HTTP URLs, or an S3 bucket. Note that publications are encrypted once: every license generated for such publication is pointing at the same encrypted file. Because these publications are stronlgy encrypted and the decryption key is secured in your SQL database, public access to these files is not problematic.  

The servers require the setup of an SQL Database. 

- SQLite is sufficient for most needs. If the "database" property of each server defines a sqlite3 driver, the db setup is dynamically achieved when the server runs for the first time. SQLite database creation scripts are also provided in the "dbmodel" folder in case they are useful. 
- MySQL, MS SQL and PostgreSQL database creation scripts are provided in the "dbmodel" folder. These scripts must be applied before launching the servers for the first time. 

Encryption Profiles
===================
Out of the box, this open-source software is using what we call the "basic" (or "test") LCP profile, i.e. a testing mode provided by the [LCP open standard](https://readium.org/lcp-specs/). Licenses generated with this "basic" profile are perfectly handled by reading applications based on [Readium Mobile](https://www.edrlab.org/software/readium-mobile/), as well as by [Thorium Reader](https://www.edrlab.org/software/thorium-reader/).

But this profile, because it is open, does not offer any security. Security is provided by a "production" profile, i.e. confidential crypto information and a personal X.509 certificate delivered to trusted implementers by [EDRLab](mailto:contact@edrlab.org). EDRLab is the wordwide LCP Certification Authority. Licenses generated with the "production" profile are handled by any LCP compliant Reading System.

Executables
===========
The server software is composed of several independant parts:

## [lcpencrypt]  

A command line utility for content encryption. This utility can be included in any processing pipeline. 

lcpencrypt can:
* Take an unprotected publication as input and generates an encrypted file as output.
* Store the encrypted file into a file system or S3 bucket.
* Notify the License server of the generation of the encrypted file.
* Optionnaly, notify the CMS of the generation of the encrypted file.

## [lcpserver]

A License server implements [Readium Licensed Content Protection 1.0](https://readium.org/lcp-specs/releases/lcp/latest).

Its functionalities can only be accessed after client authentication.

Its private functionalities are:
* Generate a license or returns a fresh license
* Update the rights associated with a license
* Get a list of licenses (optionally filtered by publication)

## [lsdserver]

A Status Server implements [Readium License Status Document 1.0](https://readium.org/lcp-specs/releases/lsd/latest).

Its public functionalities are:
* Return a license status document
* Process a device registration request
* Process a lending return request
* Process a lending renewal request

Its private functionalities (authentication required) are:
* Be notified of the generation of a new license
* Filter licenses by count of registered devices
* List all registered devices for a given license
* Revoke or cancel a license

## [frontend]

A Frontend Test Server is also provided in the project. This is a demo server we developed to provide a micro-CMS and a user interface for testing LCP licenses. It is active on https://front-prod.edrlab.org/frontend/. We do not consider it production ready, we don't update it (despite evolutions in node, npm, and the multiple node modules used a dependencies) and it will disappear in the next major version of the codebase. The Frontend Test Server MUST NOT be used in production.

This is why the installation of the Frontend Test Server is not described in the following instructions.

Install
=======

Assuming a working Go installation (*go 1.16* or higher) ...

### On Linux and MacOS: the easy route

Simply use the go install command.

```sh
# fetch, build and install the different packages and their dependencies
go install github.com/readium/readium-lcp-server/lcpencrypt@latest
go install github.com/readium/readium-lcp-server/lcpserver@latest
go install github.com/readium/readium-lcp-server/lsdserver@latest
```

"@latest" can be replaced by a specific version, e.g. "@V1.8.0" (warning: use a capital V).

You should now find the generated binaries in $GOPATH/bin: 

- `lcpencrypt`: the command line encryption tool,
- `lcpserver`: the license server,
- `lsdserver`: the status server.

### On Linux and MacOS: the developer's route

The project supports Go modules. Developers can therefore clone the codebase in the directory of their choice. This will be required to move the LCP Server to its production mode later. 

Our recommendation is to use a structure like the one below: 

```
|- <some root dir>
 |-readium
   |- readium-lcp-server  // where the codebase is cloned
   |- <config structure>  // see below, configuration 

```

Move to the `readium` directory and use:

```
git clone https://github.com/readium/readium-lcp-server.git 
```

Then compile the code with: 

```
cd readium-lcp-server
go build -o $GOPATH/bin ./lcpencrypt
go build -o $GOPATH/bin ./lcpserver
go build -o $GOPATH/bin ./lsdserver
```

You should now find the generated binaries in $GOPATH/bin (or $GOBIN if this environment variable is set).


### On Windows 10

You must first install a GCC compiler in order to compile the SQLite module and to later move to "production mode". [TDM-GCC](http://tdm-gcc.tdragon.net/download) gives great results. 

Also, in the previous instructions, replace: 

* $GOPATH with %GOPATH%
* forward slashes with backslashes in paths.

Configuration
==============

## Config structure

Our recommendation is to use a structure like the one below: 

```
|- <some root dir>
 |-readium
   |- readium-lcp-server  // where the codebase is cloned
   |- config          // where the configuration files and X509 certificates are maintained 
   |- db              // where the sqlite database files are stored (if sqlite is used) 
   |- tmp             // where temporary files are created
```

## Environment variables

The server is controlled by a yaml configuration file (e.g. "config.yaml") stored in a convenient folder, eg. `/usr/local/var/lcp`.  

The License Server and Status Server will search their configuration file in the go bin directory by default; but the path to the file should be changed using an environment variable:

* `READIUM_LCPSERVER_CONFIG` for the License server
* `READIUM_LSDSERVER_CONFIG` for the Status server

The value of the each global variable is an absolute path to the configuration file for the given server. The two servers may share the same configuration file (useful if they are executed on the same server) or each server may get its own configuration file; this is your choice. 

## Password file

The LCP and LSD servers also require authenticated API requests for some of their functionalities. A password file formatted as an Apache "htpasswd" file is used for such authentication data. The htpasswd file format is of the form:

```sh
	User:$apr1$OMWGq53X$Qf17b.ezwEM947Vrr/oTh0
```

An example of password file generator is found [here](http://www.htaccesstools.com/htpasswd-generator/). 

The password file may be shared between the LCP and LSD servers if the same credentials are used for both. The exact location and name of the file have no importance, as it will be referenced from the configuration file; but we recommand to name it `htpasswd` and place this file in the same folder as the configuration file, eg. `/usr/local/var/lcp`.

## Certificate

The License server requires an X509 certificate and its associated private key. The exact location and name of these files have no importance, as they will be referenced from the configuration file; but we recommand to keep the file name of the file provided by EDRLab and place these files in a subfolder of the previous one, eg. `/usr/local/var/lcp/cert`.

A test certificate (`cert-edrlab-test.pem`) and private key (`privkey-edrlab-test.pem`) are provided in the `test/cert` directory of the project. These files are be used as long as the LCP server is configured in test mode (`profile` = `basic`). They are replaced by a provider specific certificate and private key when the server is moved to its production mode. 

## Quick-start configuration

A quick-start configuration meant only for test purposes is available in `test/config.yaml`. This file includes a default configuration for the the LCP, LSD and frontend servers.

1. Create a `<LCP_HOME>` folder, eg. `/usr/local/var/lcp`
2. Copy the `test/config.yaml` file into LCP_HOME 
3. Replace any occurrence of `<LCP_HOME>` in config.yaml with the absolute path to the LCP_HOME folder
4. Setup the `READIUM_*_CONFIG` env variables, which must reference the configuration file
5. Generate a password file and place it into LCP_HOME
6. Create `cert`, `db`, `files/storage` folders in LCP_HOME
7. Copy the test certificate, test private key into the `cert` subfolder of LCP_HOME

## Individual server configurations

Here are the details about the configuration properties of each server. In the samples, replace `<LCP_HOME>` with the absolute path to the folder containing the configuration file, password file, encrypted files, database and certificates.

### License Server

#### profile section
`profile`: value of the LCP profile; allowed values are:
- `basic`: default value, as described in the Readium LCP specification, used for tests only.
- `1.0`: the current production profile, maintained by EDRLab.

#### lcp section
`lcp`: parameters associated with the License Server.
- `host`: the public server hostname, `hostname` by default.
- `port`: the listening port, `8989` by default.
- `public_base_url`: the URL used by the Status Server and the Frontend Test Server to communicate with this License server; combination of the host and port values on http by default.
- `auth_file`: mandatory; the path to the password file introduced in a preceding section. 
- `cert_date`: new in v1.8, a date formatted as "yyyy-mm-dd", which corresponds to the date on which a new X509 certificate has been installed on the server. This is a patch related to a temporary flaw found in several LCP compliant reading applications.
- `database`: the URI formatted connection string to the database, see models below.

Here are models for the database property (variables in curly brackets):
- sqlite: `sqlite3://file:{path-to-dot-sqlite-file}?cache=shared&mode=rwc`
- MySQL: `mysql://{username}:{password}@/{dbname}?parseTime=true`
- MS SQL Server: `mssql://server={server-address};user id={username};password={password};database={dbname}`
- PostgrSQL: `postgres://{username}:{password}@{host}:{port}/{dbname}`

Note 1 relative to MS SQL Server: when using SQL Server Express, the server-address is of type `{ip-address}\\SQLEXPRESS)`.

Note 2 relative to MS SQL Server: we've seen installs with the additional parameters `;connection timeout=30;encrypt=disable`.

Note 3 relative to PostgrSQL: add `?sslmode=disable` for a local test install. 

#### storage section
This section should be empty if the storage location of encrypted publications is managed by the lcpencrypt utility.
If this section is present and lcpencrypt does not manage the storage, all encrypted publications will be stored in the configured folder or s3 bucket.  

`storage`: parameters related to the storage of encrypted publications.
- `mode` : optional. Possible values are "fs" (default value) and "s3".

If `mode` value is `s3`, the following parameters are expected:
- `bucket` (required): name of the target S3 bucket.
- `region` (optional): name of the target AWS region.

The S3 region and client credentials default to a chain of credential providers, searched in environment variables and shared files. See [Setting up an S3 Storage](https://github.com/readium/readium-lcp-server/wiki/Setting-up-an-S3-storage) for details. 
Alternatively (but this is not recommended!), credentials can be stored in clear in the configuration file:
- `access_id`: value of the AWS access key id.
- `secret`: value of the AWS secret access key.

If `mode` value is NOT `s3`, the following paremeters are expected:
- `filesystem` subsection: parameters related to a file system storage.   
  - `directory`: absolute path of the directory in which all encrypted publications are stored. 
  - `url`: absolute http or https url of the storage volume in which all encrypted publications are stored.

#### certificate section
`certificate`: parameters related to the signature of licenses: 	
- `cert`: the path to provider certificate file (.pem or .crt). It will be inserted in the licenses and used by clients for checking the signature. 
- `private_key`: the path to the private key (.pem) asociated with the certificate. It will be used for signing licenses. 

#### license section
`license`: parameters related to static information to be included in all licenses generated by the License Server:
- `links`: subsection: links that will be included in all licenses. `hint` and `publication` links are required in a Readium LCP license.
  If no such link exists in the partial license passed from the frontend when a new license his requested, 
  these link values will be inserted in the partial license.  
  If no value is present in the configuration file and no value is inserted in the partial license, 
  the License server will reply with a 500 Server Error at license creation.
  The sub-properties of the `links` section are:
  - `status`: required, URL template; location of the Status Document associated with a License Document.
    The license identifier is inserted via the `{license_id}` variable.
    The License Status Server is expecting the following form: `https://<url>/licenses/{license_id}/status`
  - `hint`: required; location where a Reading System can redirect a user looking for additional information about the User Passphrase. 
  - `publication`: *deprecated in favor of the storage / filesystem / url parameter*, URL template; 
    Absolute http or https url of the storage volume in which all encrypted publications are stored.
    The publication identifier is inserted via the `{publication_id}` variable.

#### lsd and lsd_notify_auth section 
The License Server must be able to notify the Status Server of the generation of a new license. 

The configuration of the License Server must therefore include:

`lsd` containing `public_base_url`: the public URL of the Status Server. 

`lsd_notify_auth`: authentication parameters used by the License Server for notifying the Status Server 
of the generation of a new license. This section contains:
- `username`: required, authentication username
- `password`: required, authentication password

#### Sample config
Here is a License Server sample config:

```yaml
profile: "basic"
lcp:
    host: "192.168.0.1"
    port: 8989
    public_base_url: "http://192.168.0.1:8989/lcpserver"
    database: "sqlite3://file:/usr/local/var/readium/db/lcp.sqlite?cache=shared&mode=rwc"
    auth_file: "/usr/local/var/readium/config/htpasswd"
certificate:
    cert: "/usr/local/var/readium/config/cert.pem"
    private_key: "/usr/local/var/readium/config/privkey.pem"
license:
    links:
        status: "https://www.example.net/lsdserver/licenses/{license_id}/status"     
        hint: "https://www.example.net/static/lcp_hint.html"
lsd:
    public_base_url:  "http://192.168.0.1:8990"
lsd_notify_auth: 
    username: "adm_username"
    password: "adm_password"

```

### Status Server

#### lsd section
`lsd`: parameters associated with the Status Server. 
- `host`: the public server hostname, `hostname` by default.
- `port`: the listening port, `8990` by default.
- `public_base_url`: the URL used by the License Server to communicate with this Status Server; combination of the host and port values on http by default.
- `auth_file`: mandatory; the path to the password file introduced in a preceding section.. 
- `database`: the URI formatted connection string to the database, see above for the format.

- `license_link_url`: URL template, mandatory; this is the url from which a fresh license can be fetched from the provider's frontend server. This url template supports a `{license_id}` parameter. The final url will be inserted in the 'license' link of every status document. It must be the url of a server acting as a proxy between the user request and the License Server. Such proxy is mandatory, as the License Server  does not possess user information needed to craft a license from its identifier. If the test frontend server is used as a proxy (for tests only), the url template must be of the form "http://<frontend-server-url>/api/v1/licenses/{license_id}" (note the /api/v1 section).

#### license_status section
`license_status`: parameters related to the interactions implemented by the Status server, if any:
- `register`: boolean; if `true`, registering a device is possible; `true` by default.  
- `renew`: boolean; if `true`, loan extensions are possible; `false` by default. 
- `return`: boolean; if `true`, early returns are possible; `false` by default. 
- `renting_days`: maximum number of days allowed for a loan. The maximum license end date is based on the date the loan starts, plus this value. No loan extension is possible after this upper limit. Use a large value (20000?) if you operate a subscription model.  
- `renew_days`: default number of additional days for a loan extension. An explicit attribute of the renew command will overwrite it. 
- `renew_from_now`: boolean; if `true`, the number of days of an extension is based on the current timestamp, not the license end date. 
- `renew_expired`: boolean; if `true`, the license provider allows the extension of an expired license. 
- `renew_page_url`: URL template; if set, the renew feature is implemented as an HTML page. This url template supports a `{license_id}`, `{/license_id}` or `{?license_id}` parameter. The final url will be inserted in every status document's 'renew' link.
- `renew_custom_url`: URL template; if set, the license provider manages the renew feature. This url template supports a `{license_id}`, `{/license_id}` or `{?license_id}` parameter. The final url will be inserted in the 'renew' link of every status document.

Detailed explanations about the use of `renew_page_url` and `renew_custom_url` are found in a [specific section of the wiki](https://github.com/readium/readium-lcp-server/wiki/Integrating-the-LCP-server-with-a-content-management-system#option-manage-renew-requests-using-your-own-rules). 

#### lcp_update_auth section 
The Status Server must be able to get information from the License Server. 

The configuration of the Status Server must therefore contain:

`lcp` containing `public_base_url`: the public URL of the License Server. 

`lcp_update_auth`: authentication parameters used by the Status Server for updating a license via the License Server. This section contains:
- `username`: mandatory, authentication username
- `password`: mandatory, authentication password

#### Sample config
Here is a Status Server sample config:

```yaml
lsd:
    host: "192.168.0.1"
    port: 8990
    public_base_url: "http://192.168.0.1:8990"
    database: "sqlite3://file:/usr/local/var/lcp/db/lsd.sqlite?cache=shared&mode=rwc"
    auth_file: "/usr/local/var/lcp/htpasswd"
    license_link_url: "https://www.example.net/lcp/licenses/{license_id}"
license_status:
    register: true
    renew: true
    return: true
    renting_days: 60
    renew_days: 7

lcp:
  public_base_url:  "http://192.168.0.1:8989"
lcp_update_auth: 
    username: "adm_username"
    password: "adm_password"
```

Execution
==========
Each server must be launched in a different context (i.e. a different terminal for local use). If the path to the generated Go binaries ($GOPATH/bin) is properly set, each server can launched from any location:

- `lcpserver`
- `lsdserver`

NOTE: even if you deploy the server locally, using 127.0.0.1 is not recommended as you won't be able to access the modules from e.g. a mobile app. It is much better to use the WiFi IPv4 address of your computer and access the server from a mobile device via WiFi.  

Contributing
============
Contributions are welcome. 

Please make a Pull Request with tests at github.com/readium/readium-lcp-server
