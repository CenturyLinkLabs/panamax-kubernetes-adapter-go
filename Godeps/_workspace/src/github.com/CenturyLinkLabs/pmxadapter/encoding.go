package pmxadapter

import (
	"encoding/json"
	"log"
)

// An Encoder implements an encoding format of values to be sent as response to
// requests on the API endpoints.
type encoder interface {
	Encode(v ...interface{}) string
}

type jsonEncoder struct{}

// JsonEncoder is an Encoder that produces JSON-formatted responses.
func (jsonEncoder) Encode(v ...interface{}) string {
	var data interface{} = v
	if v == nil {
		// So that empty results produces `[]` and not `null`
		data = []interface{}{}
	} else if len(v) == 1 {
		data = v[0]
	}
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	log.Printf("%s", b)
	return string(b)
}
