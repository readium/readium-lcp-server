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

package staticapi

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"strings"

	"github.com/gorilla/mux"
	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/config"
	"github.com/readium/readium-lcp-server/frontend/webpurchase"
	"github.com/readium/readium-lcp-server/frontend/webuser"
	"github.com/readium/readium-lcp-server/license"
	"github.com/readium/readium-lcp-server/problem"
)

//DecodeJSONPurchase transforms a json string to a User struct
func DecodeJSONPurchase(r *http.Request) (webpurchase.Purchase, error) {
	var dec *json.Decoder
	if ctype := r.Header["Content-Type"]; len(ctype) > 0 && ctype[0] == api.ContentType_JSON {
		dec = json.NewDecoder(r.Body)
	}
	purchase := webpurchase.Purchase{}
	err := dec.Decode(&purchase)
	return purchase, err
}

//GetPurchasesForUser searches all purchases for a client
func GetPurchasesForUser(w http.ResponseWriter, r *http.Request, s IServer) {
	var page int64
	var perPage int64
	var err error
	var id int64
	vars := mux.Vars(r)
	if id, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// user id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	}
	if r.FormValue("page") != "" {
		page, err = strconv.ParseInt((r).FormValue("page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		page = 1
	}
	if r.FormValue("per_page") != "" {
		perPage, err = strconv.ParseInt((r).FormValue("per_page"), 10, 32)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
			return
		}
	} else {
		perPage = 30
	}
	if page > 0 {
		page-- //pagenum starting at 0 in code, but user interface starting at 1
	}
	if page < 0 {
		problem.Error(w, r, problem.Problem{Detail: "page must be positive integer"}, http.StatusBadRequest)
		return
	}
	purchases := make([]webpurchase.Purchase, 0)
	//log.Println("ListAll(" + strconv.Itoa(int(per_page)) + "," + strconv.Itoa(int(page)) + ")")
	fn := s.PurchaseAPI().GetForUser(id, int(perPage), int(page))
	for it, err := fn(); err == nil; it, err = fn() {
		purchases = append(purchases, it)
	}
	if len(purchases) > 0 {
		nextPage := strconv.Itoa(int(page) + 1)
		w.Header().Set("Link", "</users/"+vars["user_id"]+"/purchases?page="+nextPage+">; rel=\"next\"; title=\"next\"")
	}
	if page > 1 {
		previousPage := strconv.Itoa(int(page) - 1)
		w.Header().Set("Link", "</users/"+vars["user_id"]+"/purchases?page="+previousPage+">; rel=\"previous\"; title=\"previous\"")
	}
	w.Header().Set("Content-Type", api.ContentType_JSON)

	enc := json.NewEncoder(w)
	err = enc.Encode(purchases)
	if err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
}

//CreatePurchase creates a purchase in the database
func CreatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	var err error
	if purchase, err = DecodeJSONPurchase(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: "Decode JSON error: " + err.Error()}, http.StatusBadRequest)
		return
	}
	//check user
	vars := mux.Vars(r)
	var id int64
	if id, err = strconv.ParseInt(vars["user_id"], 10, 64); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "User ID must be an integer"}, http.StatusBadRequest)
	} else {
		if id != purchase.User.UserID {
			problem.Error(w, r, problem.Problem{Detail: "User ID must correpond with userID in purchase"}, http.StatusBadRequest)
		}
	}

	//purchase in PUT  data  ok
	var newID int64
	if newID, err = s.PurchaseAPI().Add(purchase); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
		return
	}
	purchase.PurchaseID = newID //get new ID from database ( potential update of license_id)
	log.Println("new purchase saved, (id=" + strconv.FormatInt(newID, 10) + ") , ask license for:")
	log.Println(purchase)

	// purchase added to db
	// in real life we would need a payment ...
	if config.Config.LcpServer.PublicBaseUrl != "" { // get/create License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_, _ = io.WriteString(pw, purchase.PartialLicense)
			pw.Close() // signal end writing partial license (POST)
		}()
		// get new license from lcpserver
		log.Println("POST " + config.Config.LcpServer.PublicBaseUrl + "/contents/" + purchase.Resource + "/licenses")
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/contents/"+purchase.Resource+"/licenses", pr)
		if config.Config.LcpUpdateAuth.Username != "" {
			req.SetBasicAuth(config.Config.LcpUpdateAuth.Username, config.Config.LcpUpdateAuth.Password)
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					// got new  license, return license
					w.WriteHeader(http.StatusCreated)
					w.Header().Set("Content-Type", api.ContentType_LCP_JSON)
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					//try to find licenseID and save it , but don't really care about result
					var lic license.License
					if err = getLicenseInfo(data, &lic); err == nil {
						purchase.LicenseID = lic.Id
						if err = s.PurchaseAPI().Update(purchase); err != nil {
							log.Println(err)
						} else {
							log.Println(" purchase with  license saved:")
							log.Println(purchase)
						}
					} else {
						log.Println("reading license from body ; error?=" + err.Error())
					}

					log.Println("license created and saved (end 200;201)  ")
					w.Write(data)
					return
				}
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + pb.Title}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}
		}
	}
}

//GetLcpResource forwards a request to obtain the encrypted content to the lcp server
// this could be an ftp server or other service instead
func GetLcpResource(w http.ResponseWriter, r *http.Request, s IServer) {
	if config.Config.LcpServer.PublicBaseUrl != "" { // get encrypted content (resource)from lcp server
		vars := mux.Vars(r)
		var lcpClient = &http.Client{
			Timeout: time.Second * 15,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		pw.Close() // signal end writing partial license (POST)
		req, err := http.NewRequest("GET", config.Config.LcpServer.PublicBaseUrl+"/contents/"+vars["content_id"], pr)
		if config.Config.LcpUpdateAuth.Username != "" {
			req.SetBasicAuth(config.Config.LcpUpdateAuth.Username, config.Config.LcpUpdateAuth.Password)
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					// got answer from lcpserver, return headers
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
						}
					}
					// and resource
					io.Copy(w, response.Body)
					return
				}
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "Resource not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + pb.Title}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}
		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}
}

//GetPurchaseLicense contacts LCP server and asks a license for the purchase using the partial license and resourceID
func GetPurchaseLicense(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["purchase_id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().Get(int64(id)); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found -> ask license in lcpserver
	if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_, _ = io.WriteString(pw, purchase.PartialLicense)
			pw.Close() // signal end writing partial license (POST)
		}()
		log.Println("POST:" + config.Config.LcpServer.PublicBaseUrl + "/contents/" + purchase.Resource + "/licenses")
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/contents/"+purchase.Resource+"/licenses", pr)

		if config.Config.LcpUpdateAuth.Username != "" {
			log.Println("login " + config.Config.LcpUpdateAuth.Username)
			req.SetBasicAuth(config.Config.LcpUpdateAuth.Username, config.Config.LcpUpdateAuth.Password)
		} else {
			log.Println("CHECK CONFIGURATION : No login for private lcp method ! ")
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					// got new  license, return license
					// get license_id and save it in the database
					w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
					w.Header().Set("Content-Disposition", "attachment; filename=\""+purchase.Label+".lcpl\"")
					io.Copy(w, response.Body)
					//try to find licenseID and save it , but don't really care about result
					if data, err := ioutil.ReadAll(response.Body); err == nil {
						var lcpLicense license.License
						if err = getLicenseInfo(data, &lcpLicense); err != nil {
							purchase.LicenseID = lcpLicense.Id
							_ = s.PurchaseAPI().Update(purchase)
						} else {
							log.Println(err)
							log.Println(response.Body)

						}

					}
					return
				}
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + pb.Title}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}
}

//GetPurchasePublication contacts LCP server and asks a license for the purchase using the partial license and resourceID
// return a publication
func GetPurchasePublication(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["purchase_id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().Get(int64(id)); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found -> ask license in lcpserver
	if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		pr, pw := io.Pipe()
		defer pr.Close()
		go func() {
			_, _ = io.WriteString(pw, purchase.PartialLicense)
			pw.Close() // signal end writing partial license (POST)
		}()
		var PostURL string
		if purchase.LicenseID != "" { // renew license
			PostURL = config.Config.LcpServer.PublicBaseUrl + "/licenses/" + purchase.LicenseID + "/publication"
		} else { // create license
			PostURL = config.Config.LcpServer.PublicBaseUrl + "/contents/" + purchase.Resource + "/publications"
		}
		log.Println("POST:" + PostURL)
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", PostURL, pr)

		if config.Config.LcpUpdateAuth.Username != "" {
			log.Println("login " + config.Config.LcpUpdateAuth.Username)
			req.SetBasicAuth(config.Config.LcpUpdateAuth.Username, config.Config.LcpUpdateAuth.Password)
		} else {
			log.Println("CHECK CONFIGURATION : No login for private lcp method ! ")
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					// got new  publication & license, return it
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
							if purchase.LicenseID == "" && name == "X-Lcp-License" {
								// get license_id and save it in the database
								purchase.LicenseID = value
								go s.PurchaseAPI().Update(purchase) // run concurrently and don't care for result, let's hope it gets saved
							}
						}
					}
					io.Copy(w, response.Body)
					return
				}
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + pb.Title}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}
}

//GetPurchase gets a purchase by its ID in the database
func GetPurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["purchase_id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	purchase.User = *new(webuser.User)
	if purchase, err = s.PurchaseAPI().Get(int64(id)); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found
	// purchase.PartialLicense = "*" //hide partialLicense?
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err == nil {
		// send json of correctly encoded user info
		w.Header().Set("Content-Type", api.ContentType_JSON)
		w.WriteHeader(http.StatusOK)
		return
	}
	problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
}

//GetPurchaseByLicenseID gets a purchase by a LicenseID in the database
func GetPurchaseByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var err error

	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["licenseID"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
	// purchase found
	enc := json.NewEncoder(w)
	if err = enc.Encode(purchase); err == nil {
		// send json of correctly encoded user info
		w.Header().Set("Content-Type", api.ContentType_JSON)
		w.WriteHeader(http.StatusOK)
		return
	}
	problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
}

// getLicenseInfo decoldes a license in data (bytes, response.body)
func getLicenseInfo(data []byte, lic *license.License) error {
	var dec *json.Decoder
	dec = json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&lic); err != nil {
		return err
	}
	return nil
}

//RenewLicenseByLicenseID searches a purchase by a LicenseID in the database, and
// contacts the tcp server in order to renew the license
func RenewLicenseByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	purchase := webpurchase.Purchase{}
	vars := mux.Vars(r)
	var err error
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["license_id"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}

	// purchase found,  get a renewed license from lcpserver
	if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		log.Println("POST " + config.Config.LcpServer.PublicBaseUrl + "/licenses/" + vars["license_id"])
		log.Println("BODY (partial license)")
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/licenses/"+vars["license_id"], strings.NewReader(purchase.PartialLicense))
		Auth := config.Config.LcpUpdateAuth
		if Auth.Username != "" {
			req.SetBasicAuth(Auth.Username, Auth.Password)
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					//forward headers
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
						}
					}
					// and license
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					w.Write(data)
					return
				}
			case 206:
				problem.Error(w, r, problem.Problem{Detail: "Partial content (invalid license) from LCP server"}, http.StatusNotFound)
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server : statuscode=" + strconv.Itoa(response.StatusCode)}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}

}

//RenewLicensePublicationByLicenseID searches a purchase by a LicenseID in the database, and
// contacts the tcp server in order to renew the license and return a publication with the license
// TODO !!!! (this actually returns the license= copy method RenewLicenseByLicenseID)
func RenewLicensePublicationByLicenseID(w http.ResponseWriter, r *http.Request, s IServer) {
	purchase := webpurchase.Purchase{}
	vars := mux.Vars(r)
	var err error
	if purchase, err = s.PurchaseAPI().GetByLicenseID(vars["license_id"]); err != nil {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}

	// purchase found,  get a renewed license from lcpserver
	if config.Config.LcpServer.PublicBaseUrl != "" { // get updated License from lcp server
		var lcpClient = &http.Client{
			Timeout: time.Second * 5,
		}
		log.Println("POST " + config.Config.LcpServer.PublicBaseUrl + "/licenses/" + vars["license_id"] + "/publication")
		log.Println("BODY (partial license)")
		log.Println(purchase.PartialLicense)
		req, err := http.NewRequest("POST", config.Config.LcpServer.PublicBaseUrl+"/licenses/"+vars["license_id"]+"/publication", strings.NewReader(purchase.PartialLicense))
		Auth := config.Config.LcpUpdateAuth
		if Auth.Username != "" {
			req.SetBasicAuth(Auth.Username, Auth.Password)
		}
		req.Header.Add("Content-Type", api.ContentType_LCP_JSON)
		response, err := lcpClient.Do(req)
		if err != nil {
			problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :" + err.Error()}, http.StatusInternalServerError)
		} else {
			defer req.Body.Close()
			defer response.Body.Close()
			switch response.StatusCode {
			case 200, 201:
				{
					//forward headers
					for name, headers := range response.Header {
						for _, value := range headers {
							w.Header().Add(name, value)
						}
					}
					// and publication/license
					data, err := ioutil.ReadAll(response.Body)
					if err != nil {
						problem.Error(w, r, problem.Problem{Detail: "Error writing response:" + err.Error()}, http.StatusInternalServerError)
					}
					w.Write(data)
					return
				}
			case 206:
				problem.Error(w, r, problem.Problem{Detail: "Partial content (invalid license) from LCP server"}, http.StatusNotFound)
			case 404:
				problem.Error(w, r, problem.Problem{Detail: "License not found on LCP server"}, http.StatusNotFound)
			default: //other error ?
				{
					var pb problem.Problem
					var dec *json.Decoder
					dec = json.NewDecoder(response.Body)
					err := dec.Decode(&pb)
					if err == nil {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server : statuscode=" + strconv.Itoa(response.StatusCode)}, http.StatusInternalServerError)
					} else {
						problem.Error(w, r, problem.Problem{Detail: "Error in LCP Server :"}, http.StatusInternalServerError)
					}
				}
			}

		}
	} else { // incorrect config
		problem.Error(w, r, problem.Problem{Detail: "No LCP server defined to contact for a new license, check your configuration!"}, http.StatusInternalServerError)
	}

}

//UpdatePurchase updates a purchase in the database
func UpdatePurchase(w http.ResponseWriter, r *http.Request, s IServer) {
	var purchase webpurchase.Purchase
	vars := mux.Vars(r)
	var id int
	var err error
	if id, err = strconv.Atoi(vars["id"]); err != nil {
		// id is not a number
		problem.Error(w, r, problem.Problem{Detail: "Purchase ID must be an integer"}, http.StatusBadRequest)
	}
	//ID is a number, check user (json)
	if purchase, err = DecodeJSONPurchase(r); err != nil {
		problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusBadRequest)
	}
	// user ok, id is a number, search purchase to update
	if _, err := s.PurchaseAPI().Get(int64(id)); err == nil {
		// purchase found
		if err := s.PurchaseAPI().Update(webpurchase.Purchase{PurchaseID: int64(id), User: webuser.User{UserID: purchase.User.UserID}, Resource: purchase.Resource, TransactionDate: purchase.TransactionDate, PartialLicense: purchase.PartialLicense}); err != nil {
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
	} else {
		switch err {
		case webpurchase.ErrNotFound:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusNotFound)
		default:
			problem.Error(w, r, problem.Problem{Detail: err.Error()}, http.StatusInternalServerError)
		}
	}
}
