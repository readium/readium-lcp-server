package apilsd

import (
	"net/http"

	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/problem"
)

var complianceTestNumber string = ""

var results = map[string]string{
	"s": "success",
	"e": "error",
}

const (
	LICENSE_STATUS        = "status"
	REGISTER_DEVICE       = "register"
	RENEW_LICENSE         = "renew"
	RETURN_LICENSE        = "return"
	CANCEL_REVOKE_LICENSE = "cancel_revoke"
)

func AddLogToFile(w http.ResponseWriter, r *http.Request, s Server) {
	testStage := r.FormValue("test_stage")
	testNumber := r.FormValue("test_number")
	testResult := r.FormValue("test_result")

	if testStage != "start" && testStage != "end" {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "You must type the regular stage of the compliance test"}, http.StatusBadRequest)
		return
	}

	if testStage == "start" {
		if len(testNumber) == 0 {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "You must type the number of compliance test"}, http.StatusBadRequest)
			return
		} else {
			complianceTestNumber = testNumber
			testResult = "-"
			writeLogToFile(testStage, testResult)
			return
		}
	} else {
		if testResult != "e" && testResult != "s" {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "You must type the result of compliance test"}, http.StatusBadRequest)
			return
		} else {
			testResult = results[testResult]
			writeLogToFile(testStage, testResult)
			complianceTestNumber = ""
		}
	}
}

func writeLogToFile(testStage string, result string) {
	logging.WriteToFile(complianceTestNumber, testStage, result)
}
