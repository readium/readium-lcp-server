package apilsd

import (
	"log"
	"net/http"

	"github.com/readium/readium-lcp-server/logging"
	"github.com/readium/readium-lcp-server/problem"
)

var complianceTestNumber string = ""

var results = map[string]string{
	"s": "success",
	"e": "error",
}

// Possible values of test stage
const (
	LICENSE_STATUS        = "status"
	REGISTER_DEVICE       = "register"
	RENEW_LICENSE         = "renew"
	RETURN_LICENSE        = "return"
	CANCEL_REVOKE_LICENSE = "cancel_revoke"
)

// AddLogToFile adds a log message to the log file
//
func AddLogToFile(w http.ResponseWriter, r *http.Request, s Server) {
	testStage := r.FormValue("test_stage")
	testNumber := r.FormValue("test_number")
	testResult := r.FormValue("test_result")

	log.Println("compliance test number " + testNumber + ", " + testStage + ", result " + testResult)

	if testStage != "start" && testStage != "end" {
		problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "The stage of the compliance test must be either 'start' or 'end'"}, http.StatusBadRequest)
		return
	}

	if testStage == "start" {
		if len(testNumber) == 0 {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "The number of compliance test cannot be null"}, http.StatusBadRequest)
		} else {
			complianceTestNumber = testNumber
			testResult = "-"
			logging.WriteToFile(complianceTestNumber, testStage, testResult)
		}
	} else {
		if testResult != "e" && testResult != "s" {
			problem.Error(w, r, problem.Problem{Type: "about:blank", Detail: "The result of compliance test must be either 'e' or 's'"}, http.StatusBadRequest)
		} else {
			testResult = results[testResult]
			logging.WriteToFile(complianceTestNumber, testStage, testResult)
			complianceTestNumber = ""
		}
	}
}
