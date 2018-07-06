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
	"fmt"
	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/validator"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/lib/views/assets"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type (
	ParamId struct {
		Id string `var:"id"`
	}

	ParamTitleAndId struct {
		Id    string `var:"id"`
		Title string `var:"title"`
	}

	ParamPaginationAndId struct {
		Id      string `var:"id"`
		Page    string `form:"page"`
		PerPage string `form:"per_page"`
	}

	// ParamPagination used to paginate listing
	ParamPagination struct {
		Page    string `form:"page"`
		PerPage string `form:"per_page"`
		Filter  string `form:"filter"`
	}

	ParamAndIndex struct {
		tag   string
		index int
		isVar bool
	}
	// RepositoryFile struct defines a file stored in a repository
	RepositoryFile struct {
		Name string
	}
)

func generateOrGetLicense(purchase *model.Purchase, server http.IServer) (*model.License, error) {
	return nil, nil
}

func collect(fnValue reflect.Value) (reflect.Type, reflect.Type, []ParamAndIndex) {

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
	var paramFields []ParamAndIndex

	if functionType.NumIn() == 0 {
		panic("Handler must have at least one argument: http.IServer")
	}
	// convention : first param is always IServer - to give access to configuration, storage, etc
	serverIfaceParam := functionType.In(0)
	if "http.IServer" != serverIfaceParam.String() {
		panic("bad handler func.")
	}

	for p := 1; p < functionType.NumIn(); p++ {
		param := functionType.In(p)
		paramName := param.Name()
		// param types should have the name starting with "Param" (e.g. "ParamPageAndDeviceID")
		if strings.HasPrefix(paramName, "Param") {
			paramType = param
			for j := 0; j < param.NumField(); j++ {
				field := param.Field(j)
				// if a field is read from muxer vars, it should have a tag FormToFields to the name of the required parameter
				varTag := field.Tag.Get("var")
				// if a field is read from muxer form, it should have a tag FormToFields to the name of the required parameter
				formTag := field.Tag.Get("form")
				if len(varTag) > 0 {
					paramFields = append(paramFields, ParamAndIndex{tag: varTag, index: j, isVar: true})
				}

				if len(formTag) > 0 {
					paramFields = append(paramFields, ParamAndIndex{tag: formTag, index: j})
				}
			}
			// Headers struct
		} else {
			if payloadType != nil {
				panic("Seems you are expecting two payloads on " + callerName + ". You should take only one.")
			}
			// convention : second param is always the json payload (which gets automatically decoded)
			switch functionType.In(p).Kind() {
			case reflect.Ptr:
				payloadType = functionType.In(p)
			}
		}
	}

	// the function must always return 2 params
	if functionType.NumOut() != 2 {
		panic("Handler has " + strconv.Itoa(functionType.NumOut()) + " returns. Must have two : *object or interface{}, and error. (while registering " + callerName + ")")
	}

	// first param returned must be Renderer (views.Renderer) or []byte (for downloads)
	if functionType.Out(0).String() != "*views.Renderer" && functionType.Out(0).String() != "[]uint8" {
		panic("bad handler func : (" + callerName + ") should return *views.Renderer as first param ( you provided -> " + functionType.Out(0).String() + ")")
	}

	// last param returned must be error
	if "error" != functionType.Out(1).String() {
		panic("bad handler func : should return error as second param")
	}

	//s.LogInfo("%s registered with %d input parameters and %d output parameters.", callerName, functionType.NumIn(), functionType.NumOut())
	return payloadType, paramType, paramFields
}

func renderError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html")
	view := views.Renderer{}
	view.SetWriter(w)
	view.AddKey("message", err.Error())
	view.AddKey("title", "Error")
	view.AddKey("pageTitle", "Error")
	view.Template("main/error.html.got")
	view.Render()
}

func makeHandler(router *mux.Router, server http.IServer, route string, fn interface{}) *mux.Route {
	// reflect on the provided handler
	fnValue := reflect.ValueOf(fn)

	// get payload, parameters and headers that will be injected
	payloadType, paramType, paramFields := collect(fnValue)

	// keeping a value of IServer to be passed on handler called
	serverValue := reflect.ValueOf(server)

	return router.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		// if the content type is form
		ctype := r.Header[http.HdrContentType]
		// Set up arguments for handler call : first argument is the IServer
		in := []reflect.Value{serverValue}

		if len(ctype) > 0 {
			// seems we're expecting a valid payload
			if payloadType != nil {

				// the most common scenario - expecting a struct
				deserializeTo := reflect.New(payloadType.Elem())
				in = append(in, deserializeTo)

				// it's  "application/x-www-form-urlencoded"
				if ctype[0] == http.ContentTypeFormUrlEncoded {
					// decode the payload
					parseErr := r.ParseForm()
					if parseErr != nil {
						renderError(w, parseErr)
						return
					}
					if err := http.FormToFields(deserializeTo, r.Form); err != nil {
						renderError(w, err)
						return
					}
					// it's "multipart/form-data"
				} else if strings.HasPrefix(ctype[0], http.ContentTypeMultipartForm) {
					parseErr := r.ParseMultipartForm(32 << 20)
					if parseErr != nil {
						renderError(w, parseErr)
						return
					}
					if err := http.FormToFields(deserializeTo, r.MultipartForm.Value); err != nil {
						renderError(w, err)
						return
					}
					var err error
					var paths []string
					// convention : if we have a multipart, for files there has to be a "Files" property which is a slice of strings.
					for _, fheaders := range r.MultipartForm.File {
						for _, hdr := range fheaders {
							if hdr.Filename == "" {
								// ignore empty files
								continue
							}
							var infile multipart.File
							// input
							if infile, err = hdr.Open(); nil != err {
								renderError(w, fmt.Errorf("Open multipart file error : %s", err.Error()))
								return
							}
							// destination
							var outfile *os.File
							filePath := server.Config().LutServer.MasterRepository + "/" + hdr.Filename
							if outfile, err = os.Create(filePath); nil != err {
								renderError(w, fmt.Errorf("Create file error : %s using path %s", err.Error(), filePath))
								return
							}
							defer outfile.Close()
							// 32K buffer copy
							if _, err = io.Copy(outfile, infile); nil != err {
								renderError(w, fmt.Errorf("Copy multipart file error : %s", err.Error()))
								return
							}
							paths = append(paths, filePath)
						}
					}
					field := deserializeTo.Elem().FieldByName("Files")
					field.Set(reflect.ValueOf(paths))
				} else {
					// it's nothing we can handle
					// read the request body
					reqBody, err := ioutil.ReadAll(r.Body)
					if err != nil {
						renderError(w, err)
						return
					}
					// defering close
					defer r.Body.Close()
					server.LogError("Content type %q not handled at the time.", ctype[0])
					renderError(w, fmt.Errorf("Not implemented : %s", reqBody))
					return
				}

				iFace := deserializeTo.Interface()
				// checking if value is implementing Validate() error
				iVal, isValidator := iFace.(validator.IValidator)
				if isValidator {
					// it does - we call validate
					err := iVal.Validate()
					if err != nil {
						renderError(w, err)
						return
					}
				}
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

		// finally, we're calling the handler with all the params
		out := fnValue.Call(in)

		// error is carrying http status and http headers - we're taking it
		if !out[1].IsNil() {
			problem, ok := out[1].Interface().(http.Problem)
			if !ok {
				w.Header().Set(http.HdrContentType, http.ContentTypeTextHtml)
				renderError(w, fmt.Errorf("Bad http.Problem"))
				return
			}
			if problem.Status == http.StatusRedirect {
				w.Header().Set(http.HdrContentType, http.ContentTypeTextHtml)
				http.Redirect(w, r, problem.Detail, problem.Status)
				return
			}
			for hdrKey := range problem.HttpHeaders {
				w.Header().Set(hdrKey, problem.HttpHeaders.Get(hdrKey))
			}
		}

		switch value := out[0].Interface().(type) {
		case []byte:
			if !out[1].IsNil() {
				problem, ok := out[1].Interface().(http.Problem)
				if ok {
					if problem.Status != 200 {
						server.LogError("Should download but error has occurred : %#v", problem)
						renderError(w, fmt.Errorf("%s", problem.Detail))
						return
					}
				} else {
					server.LogError("Should download but UNKNOWN error has occurred : %#v", out[1])
				}
			}
			// payload is already build. Important : will overwrite headers above.
			w.Write(value)
		case io.ReadCloser:
			// it's our responsibility to close the ReadCloser
			defer value.Close()
			// we have a read closer interface - headers are set above
			_, err := io.Copy(w, value)
			if err != nil {
				server.LogError("IO Error : %v", err)
			}
		default:
			w.Header().Set(http.HdrContentType, http.ContentTypeTextHtml)
			renderer, ok := out[0].Interface().(*views.Renderer)
			if ok && renderer != nil {
				renderer.SetWriter(w)
				err := renderer.Render()
				// rendering template caused an error : usefull for debugging
				if err != nil {
					renderError(w, fmt.Errorf("template rendering error : %s", err.Error()))
					return
				}
			} else {
				if out[1].IsNil() {
					renderError(w, fmt.Errorf("Bad renderer (without error). Should always have one"))
					return
				}
				problem, ok := out[1].Interface().(http.Problem)
				if !ok {
					renderError(w, fmt.Errorf("bad http.Problem"))
					return
				}
				renderError(w, fmt.Errorf("%s", problem.Detail))
				return
			}
		}
	})
}

func RegisterRoutes(muxer *mux.Router, server http.IServer) {
	// dashboard
	makeHandler(muxer, server, "/", GetIndex).Methods("GET")
	//
	// user functions
	//
	usersRoutesPathPrefix := "/users"
	makeHandler(muxer, server, usersRoutesPathPrefix, GetUsers).Methods("GET")
	makeHandler(muxer, server, usersRoutesPathPrefix, CreateOrUpdateUser).Methods("POST")
	usersRoutes := muxer.PathPrefix(usersRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(usersRoutes, server, "/{id}", GetUser).Methods("GET")
	makeHandler(usersRoutes, server, "/{id}", DeleteUser).Methods("DELETE")
	makeHandler(usersRoutes, server, "/check/{title}", CheckEmailExists).Methods("GET")
	//
	// publications
	//
	publicationsRoutesPathPrefix := "/publications"
	makeHandler(muxer, server, publicationsRoutesPathPrefix, GetPublications).Methods("GET")
	makeHandler(muxer, server, publicationsRoutesPathPrefix, CreateOrUpdatePublication).Methods("POST")
	publicationsRoutes := muxer.PathPrefix(publicationsRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(publicationsRoutes, server, "/{id}/check/{title}", CheckPublicationTitleExists).Methods("GET")
	makeHandler(publicationsRoutes, server, "/{id}", GetPublication).Methods("GET")
	makeHandler(publicationsRoutes, server, "/{id}", DeletePublication).Methods("DELETE")
	//
	// purchases
	//
	purchasesRoutesPathPrefix := "/purchases"
	makeHandler(muxer, server, purchasesRoutesPathPrefix, GetPurchases).Methods("GET")
	makeHandler(muxer, server, purchasesRoutesPathPrefix, CreateOrUpdatePurchase).Methods("POST")
	purchasesRoutes := muxer.PathPrefix(purchasesRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(purchasesRoutes, server, "/{id}", GetPurchase).Methods("GET")
	makeHandler(purchasesRoutes, server, "/{id}/license", GetPurchasedLicense).Methods("GET")
	makeHandler(purchasesRoutes, server, "/{id}", DeletePurchase).Methods("DELETE")
	makeHandler(usersRoutes, server, "/{user_id}/purchases", GetUserPurchases).Methods("GET")
	//
	// licences
	//
	licenseRoutesPathPrefix := "/licenses"
	makeHandler(muxer, server, licenseRoutesPathPrefix, GetFilteredLicenses).Methods("GET")
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(licenseRoutes, server, "/{id}", GetLicense).Methods("GET")
	makeHandler(licenseRoutes, server, "/cancel/{id}", CancelLicense).Methods("GET")
	makeHandler(licenseRoutes, server, "/revoke/{id}", RevokeLicense).Methods("GET")

	static := server.Config().LutServer.Directory
	if static == "" {
		_, file, _, _ := runtime.Caller(0)
		here := filepath.Dir(file)
		static = filepath.Join(here, "../public")
	}
	server.LogInfo("Static folder : %s", static)
	if static != "" {
		views.SetupView(server.Logger(), false, false, static)
		views.DefaultLayoutPath = "main/layout.html.got"
	} else {
		panic("Should have static folder set.")
	}

	assets.RegisterAssetRoutes(muxer)

	muxer.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		// ignoring favicon.ico request.
		if request.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusOK)
			return
		}
		server.LogError("NotFoundHandler : %s", request.URL.Path)
		// The real 404
		w.Header().Set("Content-Type", "text/html")
		view := views.Renderer{}
		view.SetWriter(w)
		view.AddKey("title", "Error 404")
		view.AddKey("message", fmt.Errorf("Requested route (%s) is not served by this server.", request.URL.Path))
		view.Template("main/error.html.got")
		view.Render()
	})
}
