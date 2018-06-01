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
	"crypto/tls"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
)

const (
	ErrorBaseUrl = "http://readium.org/license-status-document/error/"

	ServerInternalError    = ErrorBaseUrl + "server"
	RegistrationBadRequest = ErrorBaseUrl + "registration"
	ReturnBadRequest       = ErrorBaseUrl + "return"
	RenewBadRequest        = ErrorBaseUrl + "renew"
	RenewReject            = ErrorBaseUrl + "renew/date"
	CancelBadRequest       = ErrorBaseUrl + "cancel"
	FilterBadRequest       = ErrorBaseUrl + "filter"

	HdrAcceptLanguage         = "Accept-Language"
	HdrContentType            = "Content-Type"
	HdrContentLength          = "Content-Length"
	HdrContentDisposition     = "Content-Disposition"
	HdrXLcpLicense            = "X-Lcp-License"
	HdrDelay                  = "X-Add-Delay"
	HdrXContentTypeOptions    = "X-Content-Type-Options"
	HdrWWWAuthenticate        = "WWW-Authenticate"
	ContentTypeProblemJson    = "application/problem+json"
	ContentTypeLcpJson        = "application/vnd.readium.lcp.license.v1.0+json"
	ContentTypeLsdJson        = "application/vnd.readium.license.status.v1.0+json"
	ContentTypeJson           = "application/json"
	ContentTypeFormUrlEncoded = "application/x-www-form-urlencoded"
	StatusBadRequest          = http.StatusBadRequest
	StatusCreated             = http.StatusCreated
	StatusInternalServerError = http.StatusInternalServerError
	StatusNotFound            = http.StatusNotFound
	StatusOK                  = http.StatusOK
	StatusPartialContent      = http.StatusPartialContent
	StatusForbidden           = http.StatusForbidden
)

type (
	ParamAndIndex struct {
		tag   string
		index int
		isVar bool
	}
	// aliases - easy imports
	Request        = http.Request
	ResponseWriter = http.ResponseWriter

	// config
	Configuration struct {
		Certificate    Certificate        `yaml:"certificate"`
		Storage        Storage            `yaml:"storage"`
		License        License            `yaml:"license"`
		LcpServer      ServerInfo         `yaml:"lcp"`
		LsdServer      LsdServerInfo      `yaml:"lsd"`
		LutServer      FrontendServerInfo `yaml:"frontend"`
		LsdNotifyAuth  Auth               `yaml:"lsd_notify_auth"`
		LcpUpdateAuth  Auth               `yaml:"lcp_update_auth"`
		LicenseStatus  LicenseStatus      `yaml:"license_status"`
		Localization   Localization       `yaml:"localization"`
		ComplianceMode bool               `yaml:"compliance_mode"`
		GoofyMode      bool               `yaml:"goofy_mode"`
		Profile        string             `yaml:"profile,omitempty"`

		// DISABLED, see https://github.com/readium/readium-lcp-server/issues/109
		//AES256_CBC_OR_GCM string             `yaml:"aes256_cbc_or_gcm,omitempty"`
	}

	ServerInfo struct {
		Host          string `yaml:"host,omitempty"`
		Port          int    `yaml:"port,omitempty"`
		AuthFile      string `yaml:"auth_file"`
		ReadOnly      bool   `yaml:"readonly,omitempty"`
		PublicBaseUrl string `yaml:"public_base_url,omitempty"`
		Database      string `yaml:"database"`
		Directory     string `yaml:"directory,omitempty"`
	}

	LsdServerInfo struct {
		ServerInfo     `yaml:",inline"`
		LicenseLinkUrl string `yaml:"license_link_url,omitempty"`
		LogDirectory   string `yaml:"log_directory"`
	}

	FrontendServerInfo struct {
		ServerInfo          `yaml:",inline"`
		ProviderUri         string `yaml:"provider_uri"`
		RightPrint          int64  `yaml:"right_print"`
		RightCopy           int64  `yaml:"right_copy"`
		MasterRepository    string `yaml:"master_repository"`
		EncryptedRepository string `yaml:"encrypted_repository"`
	}

	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	}

	Certificate struct {
		Cert       string `yaml:"cert"`
		PrivateKey string `yaml:"private_key"`
	}

	FileSystem struct {
		Directory string `yaml:"directory"`
	}

	Storage struct {
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

	License struct {
		Links map[string]string `yaml:"links"`
	}

	LicenseStatus struct {
		Renew       bool `yaml:"renew"`
		Register    bool `yaml:"register"`
		Return      bool `yaml:"return"`
		RentingDays int  `yaml:"renting_days"`
		RenewDays   int  `yaml:"renew_days"`
	}

	Localization struct {
		Languages       []string `yaml:"languages"`
		Folder          string   `yaml:"folder"`
		DefaultLanguage string   `yaml:"default_language"`
	}

	// LcpPublication is a struct for communication with lcp-server
	LcpPublication struct {
		ContentId          string  `json:"content-id"`
		ContentKey         []byte  `json:"content-encryption-key"`
		Output             string  `json:"protected-content-location"`
		Size               *int64  `json:"protected-content-length,omitempty"`
		Checksum           *string `json:"protected-content-sha256,omitempty"`
		ContentDisposition *string `json:"protected-content-disposition,omitempty"`
		ErrorMessage       string  `json:"error"`
	}

	Problem struct {
		error       `json:"-"`
		Status      int         `json:"status,omitempty"` // if present = http response code
		Type        string      `json:"type,omitempty"`
		Title       string      `json:"title,omitempty"`
		Detail      string      `json:"detail,omitempty"`
		Instance    string      `json:"instance,omitempty"`
		HttpHeaders http.Header `json:"-"`
	}

	Server struct {
		http.Server
		Readonly       bool
		GoophyMode     bool
		Cert           *tls.Certificate
		Log            logger.StdLogger
		Cfg            Configuration
		Model          model.Store
		St             *filestor.Store
		Src            pack.ManualSource
		secretProvider SecretProvider
		realm          string
	}

	HandlerFunc func(w http.ResponseWriter, r *http.Request, s IServer)

	IServer interface {
		Config() Configuration
		Certificate() *tls.Certificate
		Source() *pack.ManualSource
		Storage() filestor.Store
		Store() model.Store
		DefaultSrvLang() string
		LogError(format string, args ...interface{})
		LogInfo(format string, args ...interface{})
		GoofyMode() bool
		NotFoundHandler() http.HandlerFunc
		HandleFunc(router *mux.Router, route string, fn interface{}, secured bool) *mux.Route
	}
)

var (
	DefaultClient = http.DefaultClient
	NewRequest    = http.NewRequest
)
