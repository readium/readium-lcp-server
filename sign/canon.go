package sign

import "encoding/json"

func Canon(in interface{}) ([]byte, error) {
	// the easiest way to canonicalize is to marshal it and reify it as a map
	// which will sort stuff correctly
	b, err := json.Marshal(in)
	if err != nil {
		return b, err
	}

	temp := new(map[string]interface{})

	err = json.Unmarshal(b, temp)
	if err != nil {
		return b, err
	}

	return json.Marshal(temp)
}
