package problem

// rfc 7807
// "application/problem+json" media type
import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/readium/readium-lcp-server/api"
	"github.com/readium/readium-lcp-server/localization"
)

func Error(w http.ResponseWriter, r *http.Request, problem api.Problem, status int) {
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

func NotFoundHandler(w http.ResponseWriter, r *http.Request, s api.Server) {
	Error(w, r, api.Problem{Type: "about:blank"}, 404)
}
