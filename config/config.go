// Copyright 2020 Readium Foundation. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file exposed on Github (readium) in the project repository.

package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Certificate    Certificate        `yaml:"certificate"`
	Storage        Storage            `yaml:"storage"`
	License        License            `yaml:"license"`
	LcpServer      ServerInfo         `yaml:"lcp"`
	LsdServer      LsdServerInfo      `yaml:"lsd"`
	FrontendServer FrontendServerInfo `yaml:"frontend"`
	LsdNotifyAuth  Auth               `yaml:"lsd_notify_auth"`
	LcpUpdateAuth  Auth               `yaml:"lcp_update_auth"`
	CMSAccessAuth  Auth               `yaml:"cms_access_auth"`
	LicenseStatus  LicenseStatus      `yaml:"license_status"`
	Localization   Localization       `yaml:"localization"`
	Logging        Logging            `yaml:"logging"`
	TestMode       bool               `yaml:"test_mode"`
	GoofyMode      bool               `yaml:"goofy_mode"`
	Profile        string             `yaml:"profile,omitempty"`

	// DISABLED, see https://github.com/readium/readium-lcp-server/issues/109
	//AES256_CBC_OR_GCM string             `yaml:"aes256_cbc_or_gcm,omitempty"`
}

type ServerInfo struct {
	Host          string `yaml:"host,omitempty"`
	Port          int    `yaml:"port,omitempty"`
	AuthFile      string `yaml:"auth_file"`
	ReadOnly      bool   `yaml:"readonly,omitempty"`
	PublicBaseUrl string `yaml:"public_base_url,omitempty"`
	Database      string `yaml:"database,omitempty"`
	CertDate      string `yaml:"cert_date,omitempty"`
	Resources     string `yaml:"resources,omitempty"`
}

type LsdServerInfo struct {
	ServerInfo     `yaml:",inline"`
	LicenseLinkUrl string `yaml:"license_link_url,omitempty"`
	UserDataUrl    string `yaml:"user_data_url,omitempty"`
}

type FrontendServerInfo struct {
	ServerInfo          `yaml:",inline"`
	Directory           string `yaml:"directory,omitempty"`
	ProviderUri         string `yaml:"provider_uri"`
	RightPrint          int32  `yaml:"right_print"`
	RightCopy           int32  `yaml:"right_copy"`
	MasterRepository    string `yaml:"master_repository"`
	EncryptedRepository string `yaml:"encrypted_repository"`
}

type Auth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Certificate struct {
	Cert       string `yaml:"cert"`
	PrivateKey string `yaml:"private_key"`
}

type FileSystem struct {
	Directory string `yaml:"directory"`
	URL       string `yaml:"url,omitempty"`
}

type Storage struct {
	FileSystem FileSystem `yaml:"filesystem"`
	AccessId   string     `yaml:"access_id"`
	DisableSSL bool       `yaml:"disable_ssl"`
	PathStyle  bool       `yaml:"path_style"`
	Mode       string     `yaml:"mode"`
	Secret     string     `yaml:"secret"`
	Endpoint   string     `yaml:"endpoint"`
	Bucket     string     `yaml:"bucket"`
	Region     string     `yaml:"region"`
	Token      string     `yaml:"token"`
}

type License struct {
	Links map[string]string `yaml:"links"`
}

type LicenseStatus struct {
	Renew          bool   `yaml:"renew"`
	Register       bool   `yaml:"register"`
	Return         bool   `yaml:"return"`
	RentingDays    int    `yaml:"renting_days"`
	RenewDays      int    `yaml:"renew_days"`
	RenewPageUrl   string `yaml:"renew_page_url,omitempty"`
	RenewCustomUrl string `yaml:"renew_custom_url,omitempty"`
	RenewExpired   bool   `yaml:"renew_expired"`
	RenewFromNow   bool   `yaml:"renew_from_now"`
}

type Localization struct {
	Languages       []string `yaml:"languages"`
	Folder          string   `yaml:"folder"`
	DefaultLanguage string   `yaml:"default_language"`
}

type Logging struct {
	Directory      string `yaml:"directory"`
	SlackToken     string `yaml:"slack_token"`
	SlackChannelID string `yaml:"slack_channel"`
}

// Config is a global variable which contains the server configuration
var Config Configuration

// ReadConfig parses the configuration file
func ReadConfig(configFileName string) {
	filename, _ := filepath.Abs(configFileName)
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		panic("Can't read config file: " + configFileName)
	}

	// Set default values
	Config.LicenseStatus.Register = true

	err = yaml.Unmarshal(yamlFile, &Config)

	if err != nil {
		panic("Can't unmarshal config. " + configFileName + " -> " + err.Error())
	}
}

// GetDatabase gets the driver name and connection string corresponding to the input
func GetDatabase(uri string) (string, string) {
	// use a sqlite memory db by default
	if uri == "" {
		uri = "sqlite3://:memory:"
	}

	parts := strings.Split(uri, "://")
	if parts[0] == "postgres" {
		return parts[0], uri
	}

	return parts[0], parts[1]
}

// SetPublicUrls sets the default publics urls of the 3 servers from the config
func SetPublicUrls() error {
	var lcpPublicBaseUrl, lsdPublicBaseUrl, frontendPublicBaseUrl, lcpHost, lsdHost, frontendHost string
	var lcpPort, lsdPort, frontendPort int
	var err error

	if lcpHost = Config.LcpServer.Host; lcpHost == "" {
		lcpHost, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if lsdHost = Config.LsdServer.Host; lsdHost == "" {
		lsdHost, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if frontendHost = Config.FrontendServer.Host; frontendHost == "" {
		frontendHost, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if lcpPort = Config.LcpServer.Port; lcpPort == 0 {
		lcpPort = 8989
	}
	if lsdPort = Config.LsdServer.Port; lsdPort == 0 {
		lsdPort = 8990
	}
	if frontendPort = Config.FrontendServer.Port; frontendPort == 0 {
		frontendPort = 80
	}

	if lcpPublicBaseUrl = Config.LcpServer.PublicBaseUrl; lcpPublicBaseUrl == "" {
		lcpPublicBaseUrl = "http://" + lcpHost + ":" + strconv.Itoa(lcpPort)
		Config.LcpServer.PublicBaseUrl = lcpPublicBaseUrl
	}
	if lsdPublicBaseUrl = Config.LsdServer.PublicBaseUrl; lsdPublicBaseUrl == "" {
		lsdPublicBaseUrl = "http://" + lsdHost + ":" + strconv.Itoa(lsdPort)
		Config.LsdServer.PublicBaseUrl = lsdPublicBaseUrl
	}
	if frontendPublicBaseUrl = Config.FrontendServer.PublicBaseUrl; frontendPublicBaseUrl == "" {
		frontendPublicBaseUrl = "http://" + frontendHost + ":" + strconv.Itoa(frontendPort)
		Config.FrontendServer.PublicBaseUrl = frontendPublicBaseUrl
	}

	return err
}
