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
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/lib/http"
	"github.com/readium/readium-lcp-server/lib/validator"
	"github.com/readium/readium-lcp-server/lib/views"
	"github.com/readium/readium-lcp-server/lib/views/assets"
	"github.com/readium/readium-lcp-server/model"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type (
	ParamId struct {
		Id string `var:"id"`
	}
	ParamTitle struct {
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
)

var ErrNotFound = errors.New("License not found")

func generateOrGetLicense(purchase *model.Purchase, server http.IServer) (*model.License, error) {

	// setFieldsFromForm the mandatory provider URI
	if server.Config().LutServer.ProviderUri == "" {
		return nil, errors.New("Mandatory provider URI missing in the configuration")
	}
	encryptedAttrs := []string{"email", "name"}
	// create a partial license
	partialLicense := model.License{
		Provider: server.Config().LutServer.ProviderUri,
		User: &model.User{
			Email:     purchase.User.Email,
			Name:      purchase.User.Name,
			UUID:      purchase.User.UUID,
			Encrypted: encryptedAttrs,
		},
	}
	// get the hashed passphrase from the purchase
	userKeyValue, err := hex.DecodeString(purchase.User.Password)

	if err != nil {
		return nil, err
	}

	userKey := model.LicenseUserKey{
		Key: model.Key{
			Algorithm: "http://www.w3.org/2001/04/xmlenc#sha256",
		},
		Hint:  purchase.User.Hint,
		Value: string(userKeyValue),
	}
	partialLicense.Encryption.UserKey = userKey

	// In case of a creation of license, add the user rights
	if purchase.LicenseUUID == nil {
		// in case of undefined conf values for copy and print rights,
		// these rights will be setFieldsFromForm to zero
		copyVal := server.Config().LutServer.RightCopy
		printVal := server.Config().LutServer.RightPrint
		userRights := model.LicenseUserRights{
			Copy:  &model.NullInt{NullInt64: sql.NullInt64{Int64: copyVal, Valid: true}},
			Print: &model.NullInt{NullInt64: sql.NullInt64{Int64: printVal, Valid: true}},
		}

		// if this is a loan, include start and end dates from the purchase info
		if purchase.Type == model.LOAN {
			userRights.Start = purchase.StartDate
			userRights.End = purchase.EndDate
		}

		partialLicense.Rights = &userRights
	}

	// encode in json
	jsonBody, err := json.Marshal(partialLicense)
	if err != nil {
		return nil, err
	}

	// get the url of the lcp server
	lcpServerConfig := server.Config().LcpServer
	var lcpURL string

	if purchase.LicenseUUID == nil || !purchase.LicenseUUID.Valid {
		// if the purchase contains no license id, generate a new license
		lcpURL = lcpServerConfig.PublicBaseUrl + "/contents/" + purchase.Publication.UUID + "/license"
	} else {
		// if the purchase contains a license id, fetch an existing license
		// note: this will not update the license rights
		lcpURL = lcpServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String
	}

	// add the partial license to the POST request
	req, err := http.NewRequest("POST", lcpURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	lcpUpdateAuth := server.Config().LcpUpdateAuth
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// the body is a partial license in json format
	req.Header.Add("Content-Type", http.ContentTypeLcpJson)

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
		server.LogError("Error POST on LCP Server : %v", err)
		return nil, err
	}

	// we have a body, defering close
	defer resp.Body.Close()

	// if the status code from the request to the lcp server
	// is neither 201 Created or 200 ok, return an internal error
	if (purchase.LicenseUUID == nil && resp.StatusCode != 201) ||
		(purchase.LicenseUUID != nil && resp.StatusCode != 200) {
		return nil, errors.New("The License Server returned an error")
	}

	// decode the full license
	fullLicense := &model.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(fullLicense)

	if err != nil {
		return nil, errors.New("Unable to decode license")
	}

	// store the license id if it was not already setFieldsFromForm
	if purchase.LicenseUUID == nil {
		purchase.LicenseUUID = &model.NullString{NullString: sql.NullString{String: fullLicense.Id, Valid: true}}
		err = updatePurchase(purchase, server)
		if err != nil {
			return fullLicense, err
		}
	}

	return fullLicense, nil
}

// Update modifies a purchase on a renew or return request
// parameters: a Purchase structure withID,	LicenseUUID, StartDate,	EndDate, Status
// EndDate may be undefined (nil), in which case the lsd server will choose the renew period
//
func updatePurchase(purchase *model.Purchase, server http.IServer) error {
	// Get the original purchase from the db
	origPurchase, err := server.Store().Purchase().Get(purchase.ID)

	if err != nil {
		return fmt.Errorf("Error : reading purchase with id %d", purchase.ID)
	}
	if origPurchase.Status != model.StatusOk {
		return errors.New("Cannot update an invalid purchase")
	}
	if purchase.Status == model.StatusToBeRenewed ||
		purchase.Status == model.StatusToBeReturned {

		if purchase.LicenseUUID == nil {
			return errors.New("Cannot return or renew a purchase when no license has been delivered")
		}

		lsdServerConfig := server.Config().LsdServer
		lsdURL := lsdServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String

		if purchase.Status == model.StatusToBeRenewed {
			lsdURL += "/renew"

			if purchase.EndDate != nil {
				lsdURL += "?end=" + purchase.EndDate.Time.Format(time.RFC3339)
			}

			// Next status if LSD raises no error
			purchase.Status = model.StatusOk
		} else if purchase.Status == model.StatusToBeReturned {
			lsdURL += "/return"

			// Next status if LSD raises no error
			purchase.Status = model.StatusOk
		}
		// prepare the request for renew or return to the license status server
		req, err := http.NewRequest("PUT", lsdURL, nil)
		if err != nil {
			return err
		}
		// setFieldsFromForm credentials
		lsdAuth := server.Config().LsdNotifyAuth
		if lsdAuth.Username != "" {
			req.SetBasicAuth(lsdAuth.Username, lsdAuth.Password)
		}
		// call the lsd server

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
			server.LogError("Error PUT on LCP Server : %v", err)
			return err
		}

		defer resp.Body.Close()

		// get the new end date from the license server

		// FIXME: really needed? heavy...
		license, err := getPartialLicense(origPurchase, server)
		if err != nil {
			return err
		}
		purchase.EndDate = license.Rights.End
	} else {
		// status is not "to be renewed"
		purchase.Status = model.StatusOk
	}
	err = server.Store().Purchase().Update(purchase)
	if err != nil {
		return errors.New("Unable to update the license id")
	}
	return nil
}

func getPartialLicense(purchase *model.Purchase, server http.IServer) (*model.License, error) {
	if purchase.LicenseUUID == nil {
		return nil, errors.New("No license has been yet delivered")
	}

	lcpServerConfig := server.Config().LcpServer
	lcpURL := lcpServerConfig.PublicBaseUrl + "/licenses/" + purchase.LicenseUUID.String
	// message to the console
	//log.Println("GET " + lcpURL)
	// prepare the request
	req, err := http.NewRequest("GET", lcpURL, nil)
	if err != nil {
		return nil, err
	}
	// setFieldsFromForm credentials
	lcpUpdateAuth := server.Config().LcpUpdateAuth
	if server.Config().LcpUpdateAuth.Username != "" {
		req.SetBasicAuth(lcpUpdateAuth.Username, lcpUpdateAuth.Password)
	}
	// send the request
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
		server.LogError("Error GET on LCP Server : %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	// the call must return 206 (partial content) because there is no input partial license
	if resp.StatusCode != 206 {
		// bad status code
		return nil, errors.New("The License Server returned an error")
	}
	// decode the license
	partialLicense := model.License{}
	var dec *json.Decoder
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&partialLicense)

	if err != nil {
		return nil, errors.New("Unable to decode the license")
	}

	return &partialLicense, nil
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
				// if a field is read from muxer vars, it should have a tag setFieldsFromForm to the name of the required parameter
				varTag := field.Tag.Get("var")
				// if a field is read from muxer form, it should have a tag setFieldsFromForm to the name of the required parameter
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

	// first param returned must be Renderer (views.Renderer)
	if functionType.Out(0).String() != "*views.Renderer" {
		panic("bad handler func : should return *views.Renderer as first param")
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
	view.AddKey("title", "Error")
	view.AddKey("message", err.Error())
	view.Template("main/error.html.got")
	view.Render()
}

func setFieldsFromForm(deserializeTo reflect.Value, fromForm url.Values) error {
	for k, v := range fromForm {
		field := deserializeTo.Elem().FieldByName(k)
		val := v[0]
		//server.LogInfo("%s = %s %#v", k, v, field)
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val == "" {
				val = "0"
			}
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as int")
			} else {
				field.SetInt(intVal)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if val == "" {
				val = "0"
			}
			uintVal, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as uint")
			} else {
				field.SetUint(uintVal)
			}
		case reflect.Bool:
			if val == "" {
				val = "false"
			}
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as boolean")
			} else {
				field.SetBool(boolVal)
			}
		case reflect.Float32:
			if val == "" {
				val = "0.0"
			}
			floatVal, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as 32-bit float")
			} else {
				field.SetFloat(floatVal)
			}
		case reflect.Float64:
			if val == "" {
				val = "0.0"
			}
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("Value could not be parsed as 64-bit float")
			} else {
				field.SetFloat(floatVal)
			}
		case reflect.String:
			field.SetString(val)
		}
	}
	return nil
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
		//if r.Method == "GET" {
		//	server.LogInfo("Route : %s Method : %s", r.URL.Path, r.Method)
		//} else {
		//	server.LogInfo("Route : %s Method : %s Content Type : %s", r.URL.Path, r.Method, ctype)
		//}
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
					if err := setFieldsFromForm(deserializeTo, r.Form); err != nil {
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
					if err := setFieldsFromForm(deserializeTo, r.MultipartForm.Value); err != nil {
						renderError(w, err)
						return
					}
					var err error
					var paths []string
					// convention : if we have a multipart, for files there has to be a "Files" property which is a slice of strings.
					for _, fheaders := range r.MultipartForm.File {
						for _, hdr := range fheaders {
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

		w.Header().Set(http.HdrContentType, http.ContentTypeTextHtml)
		// error is carrying http status and http headers - we're taking it
		if !out[1].IsNil() {
			problem, ok := out[1].Interface().(http.Problem)
			if !ok {
				renderError(w, fmt.Errorf("Bad http.Problem"))
				return
			}
			if problem.Status == http.StatusRedirect {
				http.Redirect(w, r, problem.Detail, problem.Status)
				return
			}
			w.WriteHeader(problem.Status)
			for hdrKey := range problem.HttpHeaders {
				w.Header().Set(hdrKey, problem.HttpHeaders.Get(hdrKey))
			}
		}
		// bytes are delivered as they are (downloads)
		if byts, ok := out[0].Interface().([]byte); ok {
			w.Write(byts)
			return
		}

		renderer, ok := out[0].Interface().(*views.Renderer)
		if ok && renderer != nil {
			renderer.SetWriter(w)
			err := renderer.Render()
			// rendering template caused an error : usefull for debugging
			if err != nil {
				renderer.AddKey("title", "Template "+renderer.GetTemplate()+" rendering error.")
				renderer.AddKey("message", err.Error())
				renderer.Template("main/error.html.got")
				renderer.Render()
			}
		} else {
			if out[1].IsNil() {
				renderError(w, fmt.Errorf("Bad renderer (without error). Should always have one"))
				return
			}
			problem, ok := out[1].Interface().(http.Problem)
			if !ok {
				renderError(w, fmt.Errorf("Bad http.Problem"))
				return
			}
			renderError(w, fmt.Errorf("%s", problem.Detail))
		}
	})
}

func readPagination(pg, perPg string, totalRecords int64) (int64, int64, error) {
	var err error
	var page, perPage int64

	if pg != "" {
		page, err = strconv.ParseInt(pg, 10, 64)
		if err != nil {
			return 0, 0, err
		}

	}
	if perPg != "" {
		perPage, err = strconv.ParseInt(perPg, 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}

	if page < 0 {
		return 0, 0, errors.New("page must be positive integer")
	}

	if page > 0 { // starting at 0 in code, but user interface starting at 1
		page--
	}

	if perPage == 0 {
		perPage = 30
	}

	if totalRecords < page*perPage {
		return 0, 0, errors.New("page outside known range")
	}

	return page, perPage, nil
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

	//
	//  repositories of master files
	//
	//
	makeHandler(muxer, server, "/repositories", GetRepositoryMasterFiles).Methods("GET")

	//
	// publications
	//
	publicationsRoutesPathPrefix := "/publications"
	makeHandler(muxer, server, publicationsRoutesPathPrefix, GetPublications).Methods("GET")
	makeHandler(muxer, server, publicationsRoutesPathPrefix, CreateOrUpdatePublication).Methods("POST")
	publicationsRoutes := muxer.PathPrefix(publicationsRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(publicationsRoutes, server, "/check-by-title", CheckPublicationByTitle).Methods("GET")
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
	makeHandler(usersRoutes, server, "/{user_id}/purchases", GetUserPurchases).Methods("GET")
	//
	// licences
	//
	licenseRoutesPathPrefix := "/licenses"
	makeHandler(muxer, server, licenseRoutesPathPrefix, GetFilteredLicenses).Methods("GET")
	licenseRoutes := muxer.PathPrefix(licenseRoutesPathPrefix).Subrouter().StrictSlash(false)
	makeHandler(licenseRoutes, server, "/{license_id}", GetLicense).Methods("GET")

	server.LogInfo("Initing repo manager.")
	repoManager = RepositoryManager{
		MasterRepositoryPath:    server.Config().LutServer.MasterRepository,
		EncryptedRepositoryPath: server.Config().LutServer.EncryptedRepository,
	}

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
