package problem

// rfc 7807
// "application/problem+json" media type
import (
	"encoding/json"
	"net/http"
	"strings"
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

func Error(w http.ResponseWriter, problem Problem, status int) {
	//todo add i18n
	w.Header().Add("Content-Type", "application/problem+json")
	if strings.Compare(problem.Type, "about:blank") == 0 {
		// lookup Title  statusText should match http status
		problem.Title = http.StatusText(status)
	}
	problem.Status = status
	jsonError, e := json.Marshal(problem)
	if e != nil {
		http.Error(w, "{}", problem.Status)
	}
	http.Error(w, string(jsonError), problem.Status)
}
