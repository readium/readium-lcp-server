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
	"encoding/json"
	"net/http"
	"strings"

	"fmt"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/epub"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/i18n"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/model"
	"io/ioutil"
	"reflect"
	"runtime"
	"strconv"
)

func (s *Server) GoofyMode() bool {
	return s.GoophyMode
}

// CreateDefaultLinks inits the global var DefaultLinks from config data
// ... DefaultLinks used in several places.
//
func (s *Server) CreateDefaultLinks(cfg License) {
	s.DefaultLinks = make(map[string]string)
	for key := range cfg.Links {
		s.DefaultLinks[key] = cfg.Links[key]
	}
}

// SetDefaultLinks sets a LicenseLink array from config links
//
func (s *Server) SetDefaultLinks() model.LicenseLinksCollection {
	links := make(model.LicenseLinksCollection, 0, 0)
	for key := range s.DefaultLinks {
		links = append(links, &model.LicenseLink{Href: s.DefaultLinks[key], Rel: key})
	}
	return links
}

// SetLicenseLinks sets publication and status links
// l.ContentId must have been set before the call
//
func (s *Server) SetLicenseLinks(l *model.License, c *model.Content) error {
	// set the links
	l.Links = s.SetDefaultLinks()

	for i := 0; i < len(l.Links); i++ {
		switch l.Links[i].Rel {
		// publication link
		case "publication":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{publication_id}", l.ContentId, 1)
			l.Links[i].Type = epub.ContentTypeEpub
			l.Links[i].Size = c.Length
			l.Links[i].Title = c.Location
			l.Links[i].Checksum = c.Sha256
			// status link
		case "status":
			l.Links[i].Href = strings.Replace(l.Links[i].Href, "{license_id}", l.Id, 1)
			l.Links[i].Type = ContentTypeLsdJson

		}

	}
	return nil
}

func (s *Server) Storage() filestor.Store {
	return *s.St
}

func (s *Server) Store() model.Store {
	return s.Model
}

func (s *Server) Certificate() *tls.Certificate {
	return s.Cert
}

func (s *Server) Source() *pack.ManualSource {
	return &s.Src
}

func (s *Server) Config() Configuration {
	return s.Cfg
}

func (s *Server) DefaultSrvLang() string {
	if s.Cfg.Localization.DefaultLanguage == "" {
		return "en_US"
	}
	return s.Cfg.Localization.DefaultLanguage
}

func (s *Server) LogError(format string, args ...interface{}) {
	s.Log.Errorf(format, args...)
}

func (s *Server) LogInfo(format string, args ...interface{}) {
	s.Log.Infof(format, args...)
}

func (s *Server) Error(w http.ResponseWriter, r *http.Request, problem Problem) {
	acceptLanguages := r.Header.Get("Accept-Language")

	w.Header().Set(HdrContentType, ContentTypeProblemJson)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// must come *after* w.Header().Add()/Set(), but before w.Write()
	w.WriteHeader(problem.Status)

	if problem.Type == "about:blank" || problem.Type == "" { // lookup Title  statusText should match http status
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, http.StatusText(problem.Status))
	} else {
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Title, problem.Title)
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &problem.Detail, problem.Detail)
	}
	jsonError, e := json.Marshal(problem)
	if e != nil {
		s.Log.Errorf("Error serializing problem : %v", e)
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonError)

	s.Log.Infof("Handled error : %s", string(jsonError))
}

func (s *Server) fastJsonError(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set(HdrContentType, ContentTypeProblemJson)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("{\"error\":\"" + message + "\"}"))
	s.Log.Infof("fastJsonError error : %s", message)
}

/**
How it works.

When you register a handler, you should know the following:
1. first parameter of the handling function should be always server http.IServer
2. second parameter is optional and represents the expected json payload
3. the rest of the parameters are as following :
a. url path parameters (e.g. /content/{content_id}) or url parameters (e.g. ?page=3)
b. header injections (e.g. "Accept-Language")
For both a) and b) cases you have to define structs.

For case a) : the struct name has to begin with "Param" and have tags which defines from where those values are read.
Keep in mind that the fields of the struct must always be of type "string" and you need to deal with conversion yourself.
E.g. tag `var:"content_id"` works with /content/{content_id} to retrieve the content_id (they are named the same) and `form:"page"` works with /content/{content_id}?page=3 to get the "page" parameter

For case b) : the struct name has to begin with "Headers" and have tag which defines which headers you are expecting. E.g. `hdr:"Accept-Language"`
Exception for case b) is (for now) "User-Agent" which gets injected

Convention : if the payload response is nil, then it has to be error.
So, never responde with both payload and error.
*/
func collect(s *Server, fnValue reflect.Value) (reflect.Type, reflect.Type, reflect.Type, []ParamAndIndex, []string) {

	// checking if we're registering a function, not something else
	functionType := fnValue.Type()
	if functionType.Kind() != reflect.Func {
		panic("Can only register functions.")
	}
	// getting the function name (for debugging purposes)
	fnCallerName := runtime.FuncForPC(fnValue.Pointer()).Name()
	parts := strings.Split(fnCallerName, "/")
	callerName := parts[len(parts)-1]

	// collecting injected parameters
	var payloadType reflect.Type
	var paramType reflect.Type
	var headersType reflect.Type
	var paramFields []ParamAndIndex
	var headerFields []string

	if functionType.NumIn() == 0 {
		panic("Handler must have at least one argument: http.IServer")
	}
	// convention : first param is always IServer - to give access to configuration, storage, etc
	serverIfaceParam := functionType.In(0)
	if "http.IServer" != serverIfaceParam.String() {
		s.LogError("First argument must be an http.IServer and you provided %q", serverIfaceParam.String())
		panic("bad handler func. Check logs.")
	}

	for p := 1; p < functionType.NumIn(); p++ {
		param := functionType.In(p)
		paramName := param.Name()
		// param types should have the name starting with "Param" (e.g. "ParamPageAndDeviceID")
		if strings.HasPrefix(paramName, "Param") {
			paramType = param
			for j := 0; j < param.NumField(); j++ {
				field := param.Field(j)
				// if a field is read from muxer vars, it should have a tag set to the name of the required parameter
				varTag := field.Tag.Get("var")
				// if a field is read from muxer form, it should have a tag set to the name of the required parameter
				formTag := field.Tag.Get("form")
				if len(varTag) > 0 {
					paramFields = append(paramFields, ParamAndIndex{tag: varTag, index: j, isVar: true})
				}

				if len(formTag) > 0 {
					paramFields = append(paramFields, ParamAndIndex{tag: formTag, index: j})
				}
			}
			// Headers struct
		} else if strings.HasPrefix(paramName, "Headers") {
			headersType = param
			// forced add of the user agent
			headerFields = append(headerFields, "User-Agent")
			for j := 0; j < param.NumField(); j++ {
				field := param.Field(j)
				// all headers should have hdr tag - only exception is "User-Agent" - more exceptions can be added
				hdrTag := field.Tag.Get("hdr")
				if len(hdrTag) > 0 {
					headerFields = append(headerFields, hdrTag)
				}
			}
		} else {
			if payloadType != nil {
				panic("Seems you are expecting two payloads. You should take only one.")
			}
			// convention : second param is always the json payload (which gets automatically decoded)
			switch functionType.In(p).Kind() {
			case reflect.Ptr, reflect.Map, reflect.Slice:
				payloadType = functionType.In(p)
				s.LogInfo("%s parameter is expected as payload", payloadType.Name())
			default:
				s.LogError("Second argument must be an *object, map, or slice and it's %q on %s.\n\tWill be ignored.", functionType.In(p).String(), callerName)
			}
		}
	}

	// the function must always return 2 params
	if functionType.NumOut() != 2 {
		panic("Handler has " + strconv.Itoa(functionType.NumOut()) + " returns. Must have two : *object or interface{}, and error.")
	}

	// last param returned must be error
	errorParam := functionType.Out(1)

	if "error" != errorParam.String() {
		s.LogError("return must be an error and it's %q", errorParam.String())
		panic("bad handler func. Check logs.")
	}

	s.LogInfo("%s registered with %d input parameters and %d output parameters.", callerName, functionType.NumIn(), functionType.NumOut())
	return payloadType, paramType, headersType, paramFields, headerFields
}

func (s *Server) HandleFunc(router *mux.Router, route string, fn interface{}, secured bool) *mux.Route {
	// reflect on the provided handler
	fnValue := reflect.ValueOf(fn)

	// get payload, parameters and headers that will be injected
	payloadType, paramType, headersType, paramFields, headerFields := collect(s, fnValue)

	// keeping a value of IServer to be passed on handler called
	serverValue := reflect.ValueOf(s)

	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {

		// if the route is secured, we're checking authority
		if secured {
			username := ""
			if username = s.checkAuth(r); username == "" {
				s.Log.Errorf("method=%s path=%s error=Unauthorized", r.Method, r.URL.Path)
				w.Header().Set("WWW-Authenticate", `Basic realm="`+s.realm+`"`)
				s.Error(w, r, Problem{Detail: "User or password do not match!", Status: http.StatusUnauthorized})
				return
			}
			s.Log.Infof("user=%s", username)
		}

		// if the content type is form
		ctype := r.Header[HdrContentType]
		if len(ctype) > 0 && ctype[0] == ContentTypeFormUrlEncoded {
			s.LogInfo("Form URL call. Work in progress.")
			fnBody := fn.(HandlerFunc)
			fnBody(w, r, s)
			return
		}

		// default fallback - always json

		// Set up arguments for handler call : first argument is the IServer
		in := []reflect.Value{serverValue}

		// seems we're expecting a valid json payload
		if payloadType != nil {

			// Building the deserialize value
			var deserializeTo reflect.Value
			switch payloadType.Kind() {
			case reflect.Slice, reflect.Map:
				deserializeTo = reflect.New(payloadType)
				in = append(in, deserializeTo.Elem())
			case reflect.Ptr:
				deserializeTo = reflect.New(payloadType.Elem())
				in = append(in, deserializeTo)
			}

			// defering close
			defer r.Body.Close()

			// now we read the request body
			reqBody, err := ioutil.ReadAll(r.Body)
			if err != nil {
				s.fastJsonError(w, r, err.Error())
				return
			}

			// json decode the payload
			if err = json.Unmarshal(reqBody, deserializeTo.Interface()); err != nil {
				s.fastJsonError(w, r, fmt.Sprintf("Unmarshal error: %v\nReceived from client : %v", err, string(reqBody)))
				return
			}

			// checking if value is implementing Validate() error
			iVal, isValidator := deserializeTo.Interface().(IValidator)
			if isValidator {
				// it does - we call validate
				err = iVal.Validate()
				if err != nil {
					s.fastJsonError(w, r, fmt.Sprintf("Validation error : %v", err))
					return
				}
			}
		} else {
			s.LogInfo("No payload will be injected.")
		}

		// we have parameters that need to be injected
		if paramType != nil {
			vars := mux.Vars(r)
			p := reflect.New(paramType).Elem()
			for _, pf := range paramFields {
				// if the parameter is in muxer vars
				if pf.isVar {
					p.Field(pf.index).Set(reflect.ValueOf(vars[pf.tag]))
				} else {
					// otherwise it must come from muxer form
					fv := r.FormValue(pf.tag)
					p.Field(pf.index).Set(reflect.ValueOf(fv))
				}
			}
			// adding the injected
			in = append(in, p)
		} else {
			s.LogInfo("No parameters will be injected.")
		}

		// we have headers that need to be injected
		if headersType != nil {
			h := reflect.New(headersType).Elem()
			for idx, hf := range headerFields {
				switch hf {
				case "User-Agent":
					h.Field(idx).Set(reflect.ValueOf(r.UserAgent()))
				default:
					h.Field(idx).Set(reflect.ValueOf(r.Header.Get(hf)))
				}
			}
			in = append(in, h)
		} else {
			s.LogInfo("No headers will be injected.")
		}

		// finally, we're calling the handler with all the params
		out := fnValue.Call(in)

		// header
		w.Header().Set(HdrContentType, ContentTypeJson)

		// processing return of the handler (should be payload, error)
		isError := out[0].IsNil()
		// preparing the json encoder
		enc := json.NewEncoder(w)
		// we have error
		if isError {
			s.LogError("Returning error : ")
			err := enc.Encode(out[1].Interface())
			if err != nil {
				s.fastJsonError(w, r, fmt.Sprintf("Error encoding json : %v", err))
				return
			}
		} else {
			// error is carrying http status - we're taking it
			if !out[1].IsNil() {
				problem := out[1].Interface().(Problem)
				w.WriteHeader(problem.Status)
				s.LogInfo("Collected status : %d", problem.Status)
			}
			// no error has occured - serializing payload
			err := enc.Encode(out[0].Interface())
			if err != nil {
				s.fastJsonError(w, r, fmt.Sprintf("Error encoding json : %v", err))
				return
			}

		}
	})
}

func (s *Server) InitAuth(realm string) {
	authFile := s.Cfg.LcpServer.AuthFile
	if authFile == "" {
		panic("Must have passwords file")
	}
	s.secretProvider = HtpasswdFileProvider(authFile)
}

func (s *Server) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Log.Infof("method=%s path=%s status=404", r.Method, r.URL.Path)
		s.Error(w, r, Problem{Status: http.StatusNotFound})
	}
}
