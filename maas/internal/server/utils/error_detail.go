package utils

import (
	"encoding/json"

    common "common/api/proto"    // import of common protobuf
)

// ErrorMessageToJSON converts common.ErrorMessage to JSON string
func ErrorMessageToJSON(errorMessage *common.ErrorMessage) string {

	jsonData, _ := json.Marshal(errorMessage)
	return string(jsonData)
}
