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
	Static        Static        `yaml:"static"`
	LicenseStatus LicenseStatus `yaml:"license_status"`
	Localization  Localization  `yaml:"localization"`
	Logging       Logging       `yaml:"logging"`
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
