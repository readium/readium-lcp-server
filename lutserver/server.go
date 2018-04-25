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

package lutserver

import (
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"time"

	"context"
	"github.com/claudiu/gocron"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/logger"
	"github.com/readium/readium-lcp-server/lutserver/ctrl"
	"github.com/readium/readium-lcp-server/store"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

//Server struct contains server info and  db interfaces
type (
	Server struct {
		http.Server
		readonly     bool
		cert         *tls.Certificate
		repositories ctrl.WebRepository
		log          logger.StdLogger
		store        store.Store
		config       api.Configuration
	}

	// HandlerFunc defines a function handled by the server
	HandlerFunc func(w http.ResponseWriter, r *http.Request, server ctrl.IServer)
	//HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s staticapi.IServer)

)

func (s *Server) fetchLicenseStatusesFromLSD() {
	s.log.Printf("AUTOMATION : Fetch and save all license status documents")

	url := s.Config().LsdServer.PublicBaseUrl + "/licenses"
	auth := s.Config().LsdNotifyAuth

	// prepare the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth.Username+":"+auth.Password)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// making request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	// If we got an error, and the context has been canceled, the context's error is probably more useful.
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	if err != nil {
		s.log.Printf("AUTOMATION : Error getting license statuses : %v", err)
		return
	}

	// we have a body, defering close
	defer resp.Body.Close()
	// reading body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.log.Fatalf("AUTOMATION : Error reading response body error : %v", err)
	}

	s.log.Printf("AUTOMATION : lsd server response : %v [http-status:%d]", body, resp.StatusCode)

	// clear the db
	err = s.store.License().PurgeDataBase()
	if err != nil {
		panic(err)
	}

	licenses, err := api.ReadLicensesPayloads(body)
	if err != nil {
		panic(err)
	}
	// fill the db
	err = s.store.License().BulkAdd(licenses)
	if err != nil {
		panic(err)
	}
}

// RepositoryAPI ( staticapi.IServer ) returns interface for repositories
func (s *Server) RepositoryAPI() ctrl.WebRepository {
	return s.repositories
}

func (s *Server) Store() store.Store {
	return s.store
}

func (s *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, s)
	})
}

func (s *Server) Config() api.Configuration {
	return s.config
}

func (s *Server) DefaultSrvLang() string {
	return s.config.Localization.DefaultLanguage
}

func (s *Server) LogError(format string, args ...interface{}) {
	s.log.Errorf(format, args...)
}

/*no private functions used
func (server *Server) handlePrivateFunc(router *mux.Router, route string, fn HandlerFunc, authenticator *auth.BasicAuth) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if api.CheckAuth(authenticator, w, r) {
			fn(w, r, server)
		}
	})
}
*/

// New creates a new webserver (basic user interface)
func New(
	cfg api.Configuration,
	log logger.StdLogger,
	repositoryAPI ctrl.WebRepository,
	store store.Store) *Server {

	tcpAddress := cfg.FrontendServer.Host + ":" + strconv.Itoa(cfg.FrontendServer.Port)

	staticFolderPath := cfg.FrontendServer.Directory
	if staticFolderPath == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		staticFolderPath = filepath.Join(here, "../frontend/manage")
	}

	filepathConfigJs := filepath.Join(staticFolderPath, "config.js")
	fileConfigJs, err := os.Create(filepathConfigJs)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := fileConfigJs.Close(); err != nil {
			panic(err)
		}
	}()

	configJs := `
	// This file is automatically generated, and git-ignored.
	// To ignore your local changes, use:
	// git update-index --assume-unchanged frontend/manage/config.js
	window.Config = {`
	configJs += "\n\tfrontend: {url: '" + cfg.FrontendServer.PublicBaseUrl + "' },\n"
	configJs += "\tlcp: {url: '" + cfg.LcpServer.PublicBaseUrl + "', user: '" + cfg.LcpUpdateAuth.Username + "', password: '" + cfg.LcpUpdateAuth.Password + "'},\n"
	configJs += "\tlsd: {url: '" + cfg.LsdServer.PublicBaseUrl + "', user: '" + cfg.LsdNotifyAuth.Username + "', password: '" + cfg.LsdNotifyAuth.Password + "'}\n}"

	log.Printf("manage/index.html config.js:")
	log.Printf(configJs)

	fileConfigJs.WriteString(configJs)
	log.Printf("... written in %s", filepathConfigJs)

	log.Printf("Static folder : %s", staticFolderPath)
	serverRouter := api.CreateServerRouter(staticFolderPath)

	server := &Server{
		Server: http.Server{
			Handler:        serverRouter.N,
			Addr:           tcpAddress,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		log:          log,
		config:       cfg,
		repositories: repositoryAPI,
		store:        store}

	// Cron, get license status information
	gocron.Start()
	// using Method expression instead of function
	gocron.Every(10).Minutes().Do((*Server).fetchLicenseStatusesFromLSD)

	apiURLPrefix := "/api/v1"

	//
	//  repositories of master files
	//
	repositoriesRoutesPathPrefix := apiURLPrefix + "/repositories"
	repositoriesRoutes := serverRouter.R.PathPrefix(repositoriesRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	server.handleFunc(repositoriesRoutes, "/master-files", ctrl.GetRepositoryMasterFiles).Methods("GET")
	//
	// dashboard
	//
	server.handleFunc(serverRouter.R, "/dashboardInfos", ctrl.GetDashboardInfos).Methods("GET")
	server.handleFunc(serverRouter.R, "/dashboardBestSellers", ctrl.GetDashboardBestSellers).Methods("GET")
	//
	// publications
	//
	publicationsRoutesPathPrefix := apiURLPrefix + "/publications"
	publicationsRoutes := serverRouter.R.PathPrefix(publicationsRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	server.handleFunc(serverRouter.R, publicationsRoutesPathPrefix, ctrl.GetPublications).Methods("GET")
	//
	server.handleFunc(serverRouter.R, publicationsRoutesPathPrefix, ctrl.CreatePublication).Methods("POST")
	//
	server.handleFunc(serverRouter.R, "/PublicationUpload", ctrl.UploadEPUB).Methods("POST")
	//
	server.handleFunc(publicationsRoutes, "/check-by-title", ctrl.CheckPublicationByTitle).Methods("GET")
	//
	server.handleFunc(publicationsRoutes, "/{id}", ctrl.GetPublication).Methods("GET")
	server.handleFunc(publicationsRoutes, "/{id}", ctrl.UpdatePublication).Methods("PUT")
	server.handleFunc(publicationsRoutes, "/{id}", ctrl.DeletePublication).Methods("DELETE")
	//
	// user functions
	//
	usersRoutesPathPrefix := apiURLPrefix + "/users"
	usersRoutes := serverRouter.R.PathPrefix(usersRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	server.handleFunc(serverRouter.R, usersRoutesPathPrefix, ctrl.GetUsers).Methods("GET")
	//
	server.handleFunc(serverRouter.R, usersRoutesPathPrefix, ctrl.CreateUser).Methods("POST")
	//
	server.handleFunc(usersRoutes, "/{id}", ctrl.GetUser).Methods("GET")
	server.handleFunc(usersRoutes, "/{id}", ctrl.UpdateUser).Methods("PUT")
	server.handleFunc(usersRoutes, "/{id}", ctrl.DeleteUser).Methods("DELETE")
	// get all purchases for a given user
	server.handleFunc(usersRoutes, "/{user_id}/purchases", ctrl.GetUserPurchases).Methods("GET")

	//
	// purchases
	//
	purchasesRoutesPathPrefix := apiURLPrefix + "/purchases"
	purchasesRoutes := serverRouter.R.PathPrefix(purchasesRoutesPathPrefix).Subrouter().StrictSlash(false)
	// get all purchases
	server.handleFunc(serverRouter.R, purchasesRoutesPathPrefix, ctrl.GetPurchases).Methods("GET")
	// create a purchase
	server.handleFunc(serverRouter.R, purchasesRoutesPathPrefix, ctrl.CreatePurchase).Methods("POST")
	// update a purchase
	server.handleFunc(purchasesRoutes, "/{id}", ctrl.UpdatePurchase).Methods("PUT")
	// get a purchase by purchase id
	server.handleFunc(purchasesRoutes, "/{id}", ctrl.GetPurchase).Methods("GET")
	// get a license from the associated purchase id
	server.handleFunc(purchasesRoutes, "/{id}/license", ctrl.GetPurchasedLicense).Methods("GET")
	//
	// licences
	//
	licenseRoutesPathPrefix := apiURLPrefix + "/licenses"
	licenseRoutes := serverRouter.R.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)
	//
	// get a list of licenses
	server.handleFunc(serverRouter.R, licenseRoutesPathPrefix, ctrl.GetFilteredLicenses).Methods("GET")
	// get a license by id
	server.handleFunc(licenseRoutes, "/{license_id}", ctrl.GetLicense).Methods("GET")

	return server
}
