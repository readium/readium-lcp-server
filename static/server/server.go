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

package staticserver

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/static/api"
	"github.com/readium/readium-lcp-server/static/webpurchase"
	"github.com/readium/readium-lcp-server/static/webuser"
)

//Server struct contains server info and  db interfaces
type Server struct {
	http.Server
	readonly  bool
	cert      *tls.Certificate
	users     webuser.WebUser
	purchases webpurchase.WebPurchase
}

// HandlerFunc type define a function handled by the server
type HandlerFunc func(w http.ResponseWriter, r *http.Request, s staticapi.IServer)

//type HandlerPrivateFunc func(w http.ResponseWriter, r *auth.AuthenticatedRequest, s staticapi.IServer)

// New creates a new webserver (basic user interface)
func New(bindAddr string, tplPath string, userAPI webuser.WebUser, purchaseAPI webpurchase.WebPurchase) *Server {

	sr := api.CreateServerRouter(tplPath)
	s := &Server{
		Server: http.Server{
			Handler:        sr.N,
			Addr:           bindAddr,
			WriteTimeout:   15 * time.Second,
			ReadTimeout:    15 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		users:     userAPI,
		purchases: purchaseAPI}
	//user functions
	s.handleFunc(sr.R, "/users/{email}", staticapi.GetUserByEmail).Methods("GET")
	s.handleFunc(sr.R, "/users", staticapi.GetUsers).Methods("GET")
	s.handleFunc(sr.R, "/users", staticapi.CreateUser).Methods("PUT")
	s.handleFunc(sr.R, "/users/{user_id}", staticapi.UpdateUser).Methods("POST")
	s.handleFunc(sr.R, "/users/{user_id}", staticapi.DeleteUser).Methods("DELETE")
	// purchases
	s.handleFunc(sr.R, "/users/{user_id}/purchases", staticapi.GetPurchasesForUser).Methods("GET")                      // get purchases for a user
	s.handleFunc(sr.R, "/users/{user_id}/purchases", staticapi.CreatePurchase).Methods("PUT")                           //add purchase
	s.handleFunc(sr.R, "/users/{user_id}/purchases/{purchase_id}", staticapi.GetPurchase).Methods("GET")                // get purchase
	s.handleFunc(sr.R, "/users/{user_id}/purchases/{purchase_id}", staticapi.UpdatePurchase).Methods("POST")            // update purchase
	s.handleFunc(sr.R, "/users/{user_id}/purchases/{purchase_id}/license", staticapi.GetPurchaseLicense).Methods("GET") // get lcp license for purchase

	// license update
	s.handleFunc(sr.R, "/licenses/{id}", staticapi.RenewLicenseByLicenseID).Methods("GET") // get license by licenseID
	// resources
	s.handleFunc(sr.R, "/contents/{content_id}", staticapi.GetLcpResource).Methods("GET") // proxy get resource from lcp server

	return s
}

//UserAPI ( staticapi.IServer )returns DB interface for users
func (server *Server) UserAPI() webuser.WebUser {
	return server.users
}

//PurchaseAPI ( staticapi.IServer )returns DB interface for pruchases
func (server *Server) PurchaseAPI() webpurchase.WebPurchase {
	return server.purchases
}

func (server *Server) handleFunc(router *mux.Router, route string, fn HandlerFunc) *mux.Route {
	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, server)
	})
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
