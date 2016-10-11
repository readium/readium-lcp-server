package api

const (
	ContentType_LCP_JSON = "application/vnd.readium.lcp.license.1.0+json"
	ContentType_LSD_JSON = "application/vnd.readium.license.status.v1.0+json"
	
	ContentType_JSON = "application/json"

	ContentType_FORM_URL_ENCODED = "application/x-www-form-urlencoded"
)

/*
type Problem struct {
	Type string `json:"type"`
	//optionnal
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"` //if present = http response code
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	//Additional members
}
*/
