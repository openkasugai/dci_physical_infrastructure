package utils

import (
	"encoding/json"

    common "common/api/proto"    // import of common protobuf

	"k8s.io/klog/v2"
)

// ErrorMessageToJSON converts ErrorMessage to JSON string
func ErrorMessageToJSON(errorMessage *common.ErrorMessage) string {

	jsonData, err := json.Marshal(errorMessage)
	if err != nil {
		klog.Error(err.Error())
		return ""
	}

	return string(jsonData)
}
