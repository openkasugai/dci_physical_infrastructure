package utils

import (
	"encoding/json"

	"k8s.io/klog/v2"

    common "common/api/proto"    // import of common protobuf
)

// ErrorMessageToJSON converts common.ErrorMessage to JSON string
func ErrorMessageToJSON(errorMessage *common.ErrorMessage) string {
	var result string
	defer func() {
		klog.V(2).InfoS("end ErrorMessageToJSON",
			"result", result)
	}()
	klog.V(2).InfoS("start ErrorMessageToJSON",
		"error_message", errorMessage)

	jsonData, err := json.Marshal(errorMessage)
	if err != nil {
		klog.V(2).InfoS("branch: JSON marshal failed", "error", err.Error())
		klog.Error(err.Error())
		return ""
	}

	result = string(jsonData)
	klog.V(2).InfoS("branch: JSON marshal successful")
	return result
}
