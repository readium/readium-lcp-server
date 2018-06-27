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

	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/filestor"
	"github.com/readium/readium-lcp-server/lib/i18n"
	"github.com/readium/readium-lcp-server/lib/logger"
	"github.com/readium/readium-lcp-server/lib/pack"
	"github.com/readium/readium-lcp-server/lib/validator"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"reflect"
	"runtime"
	"strconv"
)

func (s *Server) GoofyMode() bool {
	return s.GoophyMode
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

func (s *Server) Logger() logger.StdLogger {
	return s.Log
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

func (s *Server) fastJsonError(w http.ResponseWriter, r *http.Request, status int, message string) {
	acceptLanguages := r.Header.Get(HdrAcceptLanguage)

	w.Header().Set(HdrContentType, ContentTypeProblemJson)
	w.Header().Set(HdrXContentTypeOptions, "nosniff")
	w.WriteHeader(status)

	if len(acceptLanguages) > 0 {
		// TODO : test localization
		localizedMessage := ""
		i18n.LocalizeMessage(s.Cfg.Localization.DefaultLanguage, acceptLanguages, &localizedMessage, message)
		w.Write([]byte("{\"error\":\"" + localizedMessage + "\"}"))

	} else {
		w.Write([]byte("{\"error\":\"" + message + "\"}"))
	}

	s.Log.Errorf("fast Json error : %s", message)
}

/**
Small Dependency Injection for Json API - How it works:

When you register a handler, you should know the following:
1. first parameter of the handling function should be always server http.IServer
2. second parameter is optional and represents the expected json payload (which is a go struct). Sometimes, the http.Request is needed in the controller - it is provided for you
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
		s.LogError("First argument must be an http.IServer and you provided %q on registering %s", serverIfaceParam.String(), callerName)
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
				panic("Seems you are expecting two payloads on " + callerName + ". You should take only one.")
			}
			// convention : second param is always the json payload (which gets automatically decoded)
			switch functionType.In(p).Kind() {
			case reflect.Ptr, reflect.Map, reflect.Slice:
				payloadType = functionType.In(p)
			default:
				s.LogInfo("Second argument must be an *object, map, or slice and it's %q on %s [will be ignored].", functionType.In(p).String(), callerName)
			}
		}
	}

	// the function must always return 2 params
	if functionType.NumOut() != 2 {
		panic("Handler has " + strconv.Itoa(functionType.NumOut()) + " returns. Must have two : *object or interface{}, and error. (while registering " + callerName + ")")
	}

	// last param returned must be error
	errorParam := functionType.Out(1)

	if "error" != errorParam.String() {
		s.LogError("return must be an error and it's %q", errorParam.String())
		panic("bad handler func. Check logs.")
	}

	//s.LogInfo("%s registered with %d input parameters and %d output parameters.", callerName, functionType.NumIn(), functionType.NumOut())
	return payloadType, paramType, headersType, paramFields, headerFields
}

func (s *Server) HandleFunc(router *mux.Router, route string, fn interface{}, secured bool) *mux.Route {
	// reflect on the provided handler
	fnValue := reflect.ValueOf(fn)

	// get payload, parameters and headers that will be injected
	payloadType, paramType, headersType, paramFields, headerFields := collect(s, fnValue)

	// keeping a value of IServer to be passed on handler called
	serverValue := reflect.ValueOf(s)

	// sometimes controller expects the request itself - we're providing it
	isRequestInjected := false
	if payloadType != nil && payloadType.Kind() == reflect.Ptr && payloadType.Elem().Name() == "Request" {
		isRequestInjected = true
	}

	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {

		// if the route is secured, we're checking authority
		if secured {
			username := ""
			if username = s.checkAuth(r); username == "" {
				s.Log.Errorf("method=%s path=%s error=Unauthorized", r.Method, r.URL.Path)
				w.Header().Set(HdrWWWAuthenticate, `Basic realm="`+s.realm+`"`)
				s.fastJsonError(w, r, http.StatusUnauthorized, "User or password do not match!")
				return
			}
			//s.Log.Infof("user=%s", username)
		}

		var err error
		var reqBody []byte

		// if the content type is form
		ctype := r.Header[HdrContentType]
		if len(ctype) > 0 && ctype[0] == ContentTypeFormUrlEncoded {
			// TODO : test
			reqBody = bytes.NewBufferString(r.PostFormValue("data")).Bytes()
		} else if !isRequestInjected {
			// default fallback - always json - if not request injected (body reading inside controller)

			// now we read the request body
			reqBody, err = ioutil.ReadAll(r.Body)
			if err != nil {
				s.fastJsonError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			// defering close
			defer r.Body.Close()
		}

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
				if !isRequestInjected {
					// the most common scenario - expecting a struct
					deserializeTo = reflect.New(payloadType.Elem())
					in = append(in, deserializeTo)
				}
			}
			if !isRequestInjected {
				// json decode the payload
				if err = json.Unmarshal(reqBody, deserializeTo.Interface()); err != nil {
					s.fastJsonError(w, r, http.StatusBadRequest, fmt.Sprintf("Unmarshal error: %v. Received from client : `%s`", err, string(reqBody)))
					return
				}

				// checking if value is implementing Validate() error
				iVal, isValidator := deserializeTo.Interface().(validator.IValidator)
				if isValidator {
					// it does - we call validate
					err = iVal.Validate()
					if err != nil {
						s.fastJsonError(w, r, http.StatusBadRequest, fmt.Sprintf("Validation error : %v", err))
						return
					}
				}
			} else {
				// append request as it is, since body is going to be read in controller.
				in = append(in, reflect.ValueOf(r))
			}
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
		}

		// finally, we're calling the handler with all the params
		out := fnValue.Call(in)

		// processing return of the handler (should be payload, error)
		isError := out[0].IsNil()
		// we have error
		if isError {
			// header
			w.Header().Set(HdrContentType, ContentTypeJson)

			problem, ok := out[1].Interface().(Problem)
			if !ok {
				s.fastJsonError(w, r, http.StatusInternalServerError, "Error : expecting Problem, got something else.")
				return
			}
			w.WriteHeader(problem.Status)
			// preparing the json encoder
			enc := json.NewEncoder(w)
			err := enc.Encode(problem)
			if err != nil {
				s.fastJsonError(w, r, http.StatusInternalServerError, fmt.Sprintf("Error encoding json : %v", err))
				return
			}
		} else {
			// error is carrying http status and http headers - we're taking it
			if !out[1].IsNil() {
				problem, ok := out[1].Interface().(Problem)
				if !ok {
					s.fastJsonError(w, r, http.StatusInternalServerError, "Error : expecting Problem, got something else.")
					return
				}
				// Important note : status not set, comes from io.ReadCloser (doesn't allow it)
				if problem.Status > 0 {
					w.WriteHeader(problem.Status)
				}
				for hdrKey := range problem.HttpHeaders {
					//s.LogInfo("Adding header %s = %#v", hdrKey, problem.HttpHeaders.Get(hdrKey))
					w.Header().Set(hdrKey, problem.HttpHeaders.Get(hdrKey))
				}
			}

			switch value := out[0].Interface().(type) {
			case []byte:
				// payload is already build. Important : will overwrite headers above.
				w.Write(value)
			case io.ReadCloser:
				// we have a ReadCloser interface - headers are set above
				// it's our responsibility to close the ReadCloser
				defer value.Close()
				_, err = io.Copy(w, value)
				if err != nil {
					s.LogError("IO Error : %v", err)
				}
			default:
				// header
				w.Header().Set(HdrContentType, ContentTypeJson)
				// no error has occured - serializing payload
				// preparing the json encoder
				enc := json.NewEncoder(w)
				err := enc.Encode(value)
				if err != nil {
					s.fastJsonError(w, r, http.StatusInternalServerError, fmt.Sprintf("Error encoding json : %v", err))
					return
				}
			}
		}
	})
}

func (s *Server) InitAuth(realm string, authFile string) {
	if authFile == "" {
		panic("Must have passwords file")
	}
	s.secretProvider = HtpasswdFileProvider(authFile)
}

func (s *Server) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Log.Infof("method=%s path=%s status=404", r.Method, r.URL.Path)
		s.fastJsonError(w, r, http.StatusNotFound, "Requested URL is not handled by this server.")
	}
}
