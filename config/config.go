// Copyright (c) 2016 Readium Foundation
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
// 3. Neither the name of the organization nor the names of its contributors may be
//    used to endorse or promote products derived from this software without specific
//    prior written permission
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. 

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-yaml/yaml"
)

type Configuration struct {
	Certificate   Certificate   `yaml:"certificate"`
	Storage       Storage       `yaml:"storage"`
	License       License       `yaml:"license"`
	LcpServer     ServerInfo    `yaml:"lcp"`
	LsdServer     ServerInfo    `yaml:"lsd"`
	LsdNotifyAuth Auth          `yaml:"lsd_notify_auth"`
	LcpUpdateAuth Auth          `yaml:"lcp_update_auth"`
	Static        Static        `yaml:"static"`
	LicenseStatus LicenseStatus `yaml:"license_status"`
	Localization  Localization  `yaml:"localization"`
	Logging       Logging       `yaml:"logging"`
	AES256_CBC_OR_GCM string    `yaml:"aes256_cbc_or_gcm,omitempty"`
}

type ServerInfo struct {
	Host          string `yaml:"host,omitempty"`
	Port          int    `yaml:"port,omitempty"`
	AuthFile      string `yaml:"auth_file"`
	ReadOnly      bool   `yaml:"readonly,omitempty"`
	PublicBaseUrl string `yaml:"public_base_url,omitempty"`
	Database      string `yaml:"database,omitempty"`
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
}

type Static struct {
	Directory string `yaml:"directory"`
}

type Storage struct {
	FileSystem FileSystem `yaml:"filesystem"`
	AccessId   string     `yaml:"access_id"`
	DisableSSL bool       `yaml:"disable_ssl"`
	PathStyle  bool       `yaml:"path_style"`
	Mode       string
	Secret     string
	Endpoint   string
	Bucket     string
	Region     string
	Token      string
}

type License struct {
	Links map[string]string `yaml:"links"`
}

type LicenseStatus struct {
	Renew       bool `yaml:"renew"`
	Register    bool `yaml:"register"`
	Return      bool `yaml:"return"`
	RentingDays int  `yaml:"renting_days" "default 0"`
	RenewDays   int  `yaml:"renew_days" "default 0"`
}

type Localization struct {
	Languages       []string `yaml:"languages"`
	Folder          string   `yaml:"folder"`
	DefaultLanguage string   `yaml:"default_language"`
}

type Logging struct {
	LogDirectory          string `yaml:"log_directory"`
	ComplianceTestsModeOn bool   `yaml:"compliance_tests_mode_on"`
}

var Config Configuration

func ReadConfig(configFileName string) {
	filename, _ := filepath.Abs(configFileName)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic("Can't read config file: " + configFileName)
	}

	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		panic("Can't unmarshal config. " + configFileName + " -> " + err.Error())
	}
}

func SetPublicUrls() error {
	var lcpPublicBaseUrl, lsdPublicBaseUrl, lcpHost, lsdHost string
	var lcpPort, lsdPort int
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

	if lcpPort = Config.LcpServer.Port; lcpPort == 0 {
		lcpPort = 8989
	}
	if lsdPort = Config.LsdServer.Port; lsdPort == 0 {
		lsdPort = 8990
	}

	if lcpPublicBaseUrl = Config.LcpServer.PublicBaseUrl; lcpPublicBaseUrl == "" {
		lcpPublicBaseUrl = "http://" + lcpHost + ":" + strconv.Itoa(lcpPort)
		Config.LcpServer.PublicBaseUrl = lcpPublicBaseUrl
	}
	if lsdPublicBaseUrl = Config.LsdServer.PublicBaseUrl; lsdPublicBaseUrl == "" {
		lsdPublicBaseUrl = "http://" + lsdHost + ":" + strconv.Itoa(lsdPort)
		Config.LsdServer.PublicBaseUrl = lsdPublicBaseUrl
	}

	return err
}
