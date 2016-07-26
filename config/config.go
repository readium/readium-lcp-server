package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/go-yaml/yaml"
)

type Configuration struct {
	Certificate   Certificate
	Database      string
	Storage       Storage
	License       License
	PublicBaseUrl string `yaml:"public_base_url"`
	Static        Static
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
	FileSystem FileSystem `yaml:"storage"`
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
	Links map[string]string
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
		panic("Can't unmarshal config. " + err.Error())
	}
}
