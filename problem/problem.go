package problem

// rfc 7807
// problem.Type should be an URI
// for example http://readium.org/readium/[lcpserver|lsdserver]/<code>
// for standard http error messages use "about:blank" status in json equals http status
import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/readium/readium-lcp-server/localization"
	"github.com/technoweenie/grohl"
)

type Problem struct {
	Type string `json:"type"`
	//optionnal
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"` //if present = http response code
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	//Additional members
}

func Error(w http.ResponseWriter, r *http.Request, problem Problem, status int) {
	//todo add i18n
	acceptLanguages := r.Header.Get("Accept-Language")

	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if problem.Type == "about:blank" {
		// lookup Title  statusText should match http status
		localization.LocalizeMessage(acceptLanguages, &problem.Title, http.StatusText(status))

	}
	problem.Status = status
	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	fmt.Fprintln(w, string(jsonError))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	grohl.Log(grohl.Data{"404 : path": r.URL.Path})
	Error(w, r, Problem{Type: "about:blank"}, http.StatusNotFound)
}
