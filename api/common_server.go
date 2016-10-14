package api

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
	"github.com/abbot/go-http-auth"
	
	"github.com/technoweenie/grohl"

	"github.com/readium/readium-lcp-server/problem"
)

const (
	ContentType_LCP_JSON = "application/vnd.readium.lcp.license.1.0+json"
	ContentType_LSD_JSON = "application/vnd.readium.license.status.v1.0+json"
	
	ContentType_JSON = "application/json"

	ContentType_FORM_URL_ENCODED = "application/x-www-form-urlencoded"
)

type ServerRouter struct {
	R *mux.Router
	N *negroni.Negroni	
}

func CreateServerRouter(tplPath string) (ServerRouter) {
	 
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(problem.NotFoundHandler) //handle all other requests 404

	// this demonstrates a panic report
	r.HandleFunc("/panic", func(w http.ResponseWriter, req *http.Request) {
		panic("just testing. no worries.")
	})

	//n := negroni.Classic() == negroni.New(negroni.NewRecovery(), negroni.NewLogger(), negroni.NewStatic(...))
	n := negroni.New()

	// possibly useful middlewares:
	// https://github.com/jeffbmartinez/delay

	//https://github.com/urfave/negroni#recovery
	recovery := negroni.NewRecovery()
	recovery.PrintStack = true
	recovery.ErrorHandlerFunc = problem.PanicReport
	n.Use(recovery)

	//https://github.com/urfave/negroni#logger
	n.Use(negroni.NewLogger())

	n.Use(negroni.HandlerFunc(ExtraLogger))
	
	if tplPath != "" {
		//https://github.com/urfave/negroni#static
		n.Use(negroni.NewStatic(http.Dir(tplPath)))
	}

	n.Use(negroni.HandlerFunc(CORSHeaders))
	// Does not insert CORS headers as intended, depends on Origin check in the HTTP request...we want the same headers, always.
	// IMPORT "github.com/rs/cors"
	// //https://github.com/rs/cors#parameters
	// c := cors.New(cors.Options{
	// 	AllowedOrigins: []string{"*"},
	// 	AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
	// 	Debug: true,
	// })
	// n.Use(c)
	
	n.UseHandler(r)

	sr := ServerRouter{
		R: r,
		N: n,
	}
	
	return sr
}

func ExtraLogger(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	grohl.Log(grohl.Data{"method": r.Method, "path": r.URL.Path})

// before	
	next(rw, r)
// after

	// noop
}

func CORSHeaders(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	grohl.Log(grohl.Data{"CORS": "yes"})
	rw.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Add("Access-Control-Allow-Origin", "*")

// before	
	next(rw, r)
// after

	// noop
}

func CheckAuth(authenticator *auth.BasicAuth, w http.ResponseWriter, r *http.Request) (bool) {
	var username string
	if username = authenticator.CheckAuth(r); username == "" {
		grohl.Log(grohl.Data{"error": "Unauthorized", "method": r.Method, "path": r.URL.Path})
		w.Header().Set("WWW-Authenticate", `Basic realm="`+authenticator.Realm+`"`)
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "User or password do not match!"}, http.StatusUnauthorized)
		return false
	}
	grohl.Log(grohl.Data{"user": username})
	return true
}
