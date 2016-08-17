package problem

// rfc 7807
// "application/problem+json" media type
import (
	"encoding/json"

	"net/http"
	"strings"

	//"github.com/nicksnyder/go-i18n/i18n"
	"github.com/readium/readium-lcp-server/localization"
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

	w.Header().Add("Content-Type", "application/problem+json")
	if strings.Compare(problem.Type, "about:blank") == 0 {
		// lookup Title  statusText should match http status
		localization.LocalizeMessage(acceptLanguages, &problem.Title, http.StatusText(status))

	}
	problem.Status = status
	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	http.Error(w, string(jsonError), problem.Status)
}
