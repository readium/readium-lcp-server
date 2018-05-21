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

package common

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

/*
 SecretProvider is used by authenticators. Takes user name and realm
 as an argument, returns secret required for authentication (HA1 for
 digest authentication, properly encrypted password for basic).

 Returning an empty string means failing the authentication.
*/
type SecretProvider func(user, realm string) string

/*
 Common functions for file auto-reloading
*/
type File struct {
	Path string
	Info os.FileInfo
	/* must be set in inherited types during initialization */
	Reload func()
	mu     sync.Mutex
}

func (f *File) ReloadIfNeeded() {
	info, err := os.Stat(f.Path)
	if err != nil {
		panic(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Info == nil || f.Info.ModTime() != info.ModTime() {
		f.Info = info
		f.Reload()
	}
}

/*
 Structure used for htdigest file authentication. Users map users to
 their salted encrypted password
*/
type HtpasswdFile struct {
	File
	Users map[string]string
	mu    sync.RWMutex
}

func reload_htpasswd(h *HtpasswdFile) {
	r, err := os.Open(h.Path)
	if err != nil {
		panic(err)
	}
	csv_reader := csv.NewReader(r)
	csv_reader.Comma = ':'
	csv_reader.Comment = '#'
	csv_reader.TrimLeadingSpace = true

	records, err := csv_reader.ReadAll()
	if err != nil {
		panic(err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.Users = make(map[string]string)
	for _, record := range records {
		h.Users[record[0]] = record[1]
	}
}

/*
 SecretProvider implementation based on htpasswd-formated files. Will
 reload htpasswd file on changes. Will panic on syntax errors in
 htpasswd files. Realm argument of the SecretProvider is ignored.
*/
func HtpasswdFileProvider(filename string) SecretProvider {
	h := &HtpasswdFile{File: File{Path: filename}}
	h.Reload = func() { reload_htpasswd(h) }
	return func(user, realm string) string {
		h.ReloadIfNeeded()
		h.mu.RLock()
		password, exists := h.Users[user]
		h.mu.RUnlock()
		if !exists {
			return ""
		}
		return password
	}
}

// ====

type compareFunc func(hashedPassword, password []byte) error

var (
	errMismatchedHashAndPassword = errors.New("mismatched hash and password")

	compareFuncs = []struct {
		prefix  string
		compare compareFunc
	}{
		{"", compareMD5HashAndPassword}, // default compareFunc
		{"{SHA}", compareShaHashAndPassword},
		// Bcrypt is complicated. According to crypt(3) from
		// crypt_blowfish version 1.3 (fetched from
		// http://www.openwall.com/crypt/crypt_blowfish-1.3.tar.gz), there
		// are three different has prefixes: "$2a$", used by versions up
		// to 1.0.4, and "$2x$" and "$2y$", used in all later
		// versions. "$2a$" has a known bug, "$2x$" was added as a
		// migration path for systems with "$2a$" prefix and still has a
		// bug, and only "$2y$" should be used by modern systems. The bug
		// has something to do with handling of 8-bit characters. Since
		// both "$2a$" and "$2x$" are deprecated, we are handling them the
		// same way as "$2y$", which will yield correct results for 7-bit
		// character passwords, but is wrong for 8-bit character
		// passwords. You have to upgrade to "$2y$" if you want sant 8-bit
		// character password support with bcrypt. To add to the mess,
		// OpenBSD 5.5. introduced "$2b$" prefix, which behaves exactly
		// like "$2y$" according to the same source.
		{"$2a$", bcrypt.CompareHashAndPassword},
		{"$2b$", bcrypt.CompareHashAndPassword},
		{"$2x$", bcrypt.CompareHashAndPassword},
		{"$2y$", bcrypt.CompareHashAndPassword},
	}
)

type BasicAuth struct {
	Realm   string
	Secrets SecretProvider
	// Headers used by authenticator. Set to ProxyHeaders to use with
	// proxy server. When nil, NormalHeaders are used.
	Headers *Headers
}

type AuthenticatorInterface interface {
	// NewContext returns a new context carrying authentication
	// information extracted from the request.
	NewContext(ctx context.Context, r *http.Request) context.Context

	// Wrap returns an http.HandlerFunc which wraps
	// AuthenticatedHandlerFunc with this authenticator's
	// authentication checks.
	Wrap(AuthenticatedHandlerFunc) http.HandlerFunc
}

// check that BasicAuth implements AuthenticatorInterface
var _ = (AuthenticatorInterface)((*BasicAuth)(nil))

/*
 Checks the username/password combination from the request. Returns
 either an empty string (authentication failed) or the name of the
 authenticated user.

 Supports MD5 and SHA1 password entries
*/
func (a *BasicAuth) CheckAuth(r *http.Request) string {
	s := strings.SplitN(r.Header.Get(a.Headers.V().Authorization), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return ""
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return ""
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return ""
	}
	user, password := pair[0], pair[1]
	secret := a.Secrets(user, a.Realm)
	if secret == "" {
		return ""
	}
	compare := compareFuncs[0].compare
	for _, cmp := range compareFuncs[1:] {
		if strings.HasPrefix(secret, cmp.prefix) {
			compare = cmp.compare
			break
		}
	}
	if compare([]byte(secret), []byte(password)) != nil {
		return ""
	}
	return pair[0]
}

func compareShaHashAndPassword(hashedPassword, password []byte) error {
	d := sha1.New()
	d.Write(password)
	if subtle.ConstantTimeCompare(hashedPassword[5:], []byte(base64.StdEncoding.EncodeToString(d.Sum(nil)))) != 1 {
		return errMismatchedHashAndPassword
	}
	return nil
}

const itoa64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var md5_crypt_swaps = [16]int{12, 6, 0, 13, 7, 1, 14, 8, 2, 15, 9, 3, 5, 10, 4, 11}

/*
 MD5 password crypt implementation
*/
func MD5Crypt(password, salt, magic []byte) []byte {
	d := md5.New()

	d.Write(password)
	d.Write(magic)
	d.Write(salt)

	d2 := md5.New()
	d2.Write(password)
	d2.Write(salt)
	d2.Write(password)

	for i, mixin := 0, d2.Sum(nil); i < len(password); i++ {
		d.Write([]byte{mixin[i%16]})
	}

	for i := len(password); i != 0; i >>= 1 {
		if i&1 == 0 {
			d.Write([]byte{password[0]})
		} else {
			d.Write([]byte{0})
		}
	}

	final := d.Sum(nil)

	for i := 0; i < 1000; i++ {
		d2 := md5.New()
		if i&1 == 0 {
			d2.Write(final)
		} else {
			d2.Write(password)
		}

		if i%3 != 0 {
			d2.Write(salt)
		}

		if i%7 != 0 {
			d2.Write(password)
		}

		if i&1 == 0 {
			d2.Write(password)
		} else {
			d2.Write(final)
		}
		final = d2.Sum(nil)
	}

	result := make([]byte, 0, 22)
	v := uint(0)
	bits := uint(0)
	for _, i := range md5_crypt_swaps {
		v |= (uint(final[i]) << bits)
		for bits = bits + 8; bits > 6; bits -= 6 {
			result = append(result, itoa64[v&0x3f])
			v >>= 6
		}
	}
	result = append(result, itoa64[v&0x3f])

	return append(append(append(magic, salt...), '$'), result...)
}

func compareMD5HashAndPassword(hashedPassword, password []byte) error {
	parts := bytes.SplitN(hashedPassword, []byte("$"), 4)
	if len(parts) != 4 {
		return errMismatchedHashAndPassword
	}
	magic := []byte("$" + string(parts[1]) + "$")
	salt := parts[2]
	if subtle.ConstantTimeCompare(hashedPassword, MD5Crypt(password, salt, magic)) != 1 {
		return errMismatchedHashAndPassword
	}
	return nil
}

var (
	// NormalHeaders are the regular Headers used by an HTTP Server for
	// request authentication.
	NormalHeaders = &Headers{
		Authenticate:      "WWW-Authenticate",
		Authorization:     "Authorization",
		AuthInfo:          "Authentication-Info",
		UnauthCode:        http.StatusUnauthorized,
		UnauthContentType: "text/plain",
		UnauthResponse:    fmt.Sprintf("%d %s\n", http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)),
	}
)

// Headers contains header and error codes used by authenticator.
type Headers struct {
	Authenticate      string // WWW-Authenticate
	Authorization     string // Authorization
	AuthInfo          string // Authentication-Info
	UnauthCode        int    // 401
	UnauthContentType string // text/plain
	UnauthResponse    string // Unauthorized.
}

// V returns NormalHeaders when h is nil, or h otherwise. Allows to
// use uninitialized *Headers values in structs.
func (h *Headers) V() *Headers {
	if h == nil {
		return NormalHeaders
	}
	return h
}

/*
 Request handlers must take AuthenticatedRequest instead of http.Request
*/
type AuthenticatedRequest struct {
	http.Request
	/*
	 Authenticated user name. Current API implies that Username is
	 never empty, which means that authentication is always done
	 before calling the request handler.
	*/
	Username string
}

/*
 AuthenticatedHandlerFunc is like http.HandlerFunc, but takes
 AuthenticatedRequest instead of http.Request
*/
type AuthenticatedHandlerFunc func(http.ResponseWriter, *AuthenticatedRequest)

/*
 Authenticator wraps an AuthenticatedHandlerFunc with
 authentication-checking code.

 Typical Authenticator usage is something like:

   authenticator := SomeAuthenticator(...)
   http.HandleFunc("/", authenticator(my_handler))

 Authenticator wrapper checks the user authentication and calls the
 wrapped function only after authentication has succeeded. Otherwise,
 it returns a handler which initiates the authentication procedure.
*/
type Authenticator func(AuthenticatedHandlerFunc) http.HandlerFunc

// Info contains authentication information for the request.
type Info struct {
	// Authenticated is set to true when request was authenticated
	// successfully, i.e. username and password passed in request did
	// pass the check.
	Authenticated bool

	// Username contains a user name passed in the request when
	// Authenticated is true. It's value is undefined if Authenticated
	// is false.
	Username string

	// ResponseHeaders contains extra headers that must be set by server
	// when sending back HTTP response.
	ResponseHeaders http.Header
}

/*
 http.Handler for BasicAuth which initiates the authentication process
 (or requires reauthentication).
*/

const contentType = "Content-Type"

func (a *BasicAuth) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentType, a.Headers.V().UnauthContentType)
	w.Header().Set(a.Headers.V().Authenticate, `Basic realm="`+a.Realm+`"`)
	w.WriteHeader(a.Headers.V().UnauthCode)
	w.Write([]byte(a.Headers.V().UnauthResponse))
}

/*
 BasicAuthenticator returns a function, which wraps an
 AuthenticatedHandlerFunc converting it to http.HandlerFunc. This
 wrapper function checks the authentication and either sends back
 required authentication headers, or calls the wrapped function with
 authenticated username in the AuthenticatedRequest.
*/
func (a *BasicAuth) Wrap(wrapped AuthenticatedHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if username := a.CheckAuth(r); username == "" {
			a.RequireAuth(w, r)
		} else {
			ar := &AuthenticatedRequest{Request: *r, Username: username}
			wrapped(w, ar)
		}
	}
}

type key int // used for context keys

var infoKey key = 0

// NewContext returns a context carrying authentication information for the request.
func (a *BasicAuth) NewContext(ctx context.Context, r *http.Request) context.Context {
	info := &Info{Username: a.CheckAuth(r), ResponseHeaders: make(http.Header)}
	info.Authenticated = (info.Username != "")
	if !info.Authenticated {
		info.ResponseHeaders.Set(a.Headers.V().Authenticate, `Basic realm="`+a.Realm+`"`)
	}
	return context.WithValue(ctx, infoKey, info)
}

func NewBasicAuthenticator(realm string, secrets SecretProvider) *BasicAuth {
	return &BasicAuth{Realm: realm, Secrets: secrets}
}
