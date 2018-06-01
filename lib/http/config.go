/*
 * Copyright (c) 2016-2018 Readium Foundation
 *
 * Redistribution and use in source and binary forms, with or without modification,
 * are permitted provided that the following conditions are met:
 *
 *  1. Redistributions of source code must retain the above copyright notice, this
 *     list of conditions and the following disclaimer.
 *  2. Redistributions in binary form must reproduce the above copyright notice,
 *     this list of conditions and the following disclaimer in the documentation and/or
 *     other materials provided with the distribution.
 *  3. Neither the name of the organization nor the names of its contributors may be
 *     used to endorse or promote products derived from this software without specific
 *     prior written permission
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 *  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 *  WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 *  DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 *  ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 *  (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 *  LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 *  ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 *  SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package http

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"fmt"

	"gopkg.in/yaml.v2"
)

func ReadConfig(configFileName string) (Configuration, error) {
	var Config Configuration

	filename, _ := filepath.Abs(configFileName)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		projectPath, pErr := os.Getwd()
		if pErr != nil {
			fmt.Printf("Error reading working dir : %s", pErr)
			os.Exit(1)
		}
		return Config, fmt.Errorf("Can't read config file: " + configFileName + " from " + projectPath)
	}

	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		return Config, fmt.Errorf("Can't unmarshal config. " + configFileName + " -> " + err.Error())
	}

	// was SetPublicUrls()
	var lcpHost, lsdHost, frontendHost string
	var lcpPort, lsdPort, frontendPort int

	if lcpHost = Config.LcpServer.Host; lcpHost == "" {
		lcpHost, err = os.Hostname()
		if err != nil {
			return Config, fmt.Errorf("%v", err)
		}
	}

	if lsdHost = Config.LsdServer.Host; lsdHost == "" {
		lsdHost, err = os.Hostname()
		if err != nil {
			return Config, fmt.Errorf("%v", err)
		}
	}

	if frontendHost = Config.LutServer.Host; frontendHost == "" {
		frontendHost, err = os.Hostname()
		if err != nil {
			return Config, fmt.Errorf("%v", err)
		}
	}

	if lcpPort = Config.LcpServer.Port; lcpPort == 0 {
		lcpPort = 8989
	}
	if lsdPort = Config.LsdServer.Port; lsdPort == 0 {
		lsdPort = 8990
	}
	if frontendPort = Config.LutServer.Port; frontendPort == 0 {
		frontendPort = 80
	}

	if len(Config.LcpServer.PublicBaseUrl) == 0 {
		Config.LcpServer.PublicBaseUrl = "http://" + lcpHost + ":" + strconv.Itoa(lcpPort)
	}
	if len(Config.LsdServer.PublicBaseUrl) == 0 {
		Config.LsdServer.PublicBaseUrl = "http://" + lsdHost + ":" + strconv.Itoa(lsdPort)
	}
	if len(Config.LutServer.PublicBaseUrl) == 0 {
		Config.LutServer.PublicBaseUrl = "http://" + frontendHost + ":" + strconv.Itoa(frontendPort)
	}

	return Config, nil
}
