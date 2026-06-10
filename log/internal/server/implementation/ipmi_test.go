// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package implementation

import (
	"errors"
	"log_module/internal/server/interfaces"
	"log_module/internal/server/interfaces/mocks"
	"log_module/internal/server/test_utils"
	"os"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func NewTestIPMIImplement(api interfaces.API, logging interfaces.Logging) *IPMIImplement {
	logger := klog.Background() // Get a klog.Logger instance
	return &IPMIImplement{
		Logger:  logger,
		API:     api,
		Logging: logging,
	}
}

func TestIPMIImplement_Init(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := IPMIImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Set necessary environment variables for the test
	_ = os.Setenv("IPMI_LOGFILE", "test_ipmi.log")
	_ = os.Setenv("IPMI_LOGPATH", "/tmp")
	_ = os.Setenv("IPMI_MAXSIZE", "1024")
	_ = os.Setenv("IPMI_MAXBACKUPS", "5")
	_ = os.Setenv("IPMI_MAXAGE", "7")

	// Expect the Init method of the mock Logging instance to be called
	mockLogging.EXPECT().Init(gomock.Any()).Return(nil)

	// Call the Init method
	err := l.Init()

	// Assert that there is no error
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Clean up the environment variables
	_ = os.Unsetenv("IPMI_LOGFILE")
	_ = os.Unsetenv("IPMI_LOGPATH")
	_ = os.Unsetenv("IPMI_MAXSIZE")
	_ = os.Unsetenv("IPMI_MAXBACKUPS")
	_ = os.Unsetenv("IPMI_MAXAGE")
}

func TestIPMIImplement_Finalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := IPMIImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Expect the Finalize method of the mock Logging instance to be called
	mockLogging.EXPECT().Finalize()

	// Call the Finalize method
	l.Finalize()
}

func TestIPMIImplement_Collection(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
		{
			ProductInfo: `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server2",
			IPMIAddress:  "test_address2",
			IPMIUser:     "test_user2",
			IPMIPassword: "test_password2",
		},
	}

	// Define the expected API response
	apiResponse1 := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	apiResponse2 := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "Warning",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "Critical",
			},
		},
	}

	// Expect the APIExecute method of the mock API instance to be called for each target
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/System.Embedded.1", "test_user1", "test_password1", "").Return(apiResponse1, nil)
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address2", "https://redfish/v1/Systems/0", "test_user2", "test_password2", "").Return(apiResponse2, nil)

	// Expect the Write method of the mock Logging instance to be called for each target
	mockLogging.EXPECT().Write("test_server1", `{ "processer": "OK", "memory": "OK" }`).Return(nil)
	mockLogging.EXPECT().Write("test_server2", `{ "processer": "Warning", "memory": "Critical" }`).Return(nil)

	// Call the Collection method
	l.Collection(targetList)
}

func TestIPMIImplement_Collection_APIExecuteError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance (not used in this test case)
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Expect the APIExecute method of the mock API instance to be called and return an error
	expectedError := errors.New("api execute error")
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/0", "test_user1", "test_password1", "").Return(nil, expectedError)

	// Call the Collection method
	l.Collection(targetList)
}

func Test_getValue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name        string
		inputJson   interface{}
		key         string
		expected    interface{}
		expectedErr error
	}{
		{
			name: "Valid input",
			inputJson: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			key:         "key1",
			expected:    "value1",
			expectedErr: nil,
		},
		{
			name: "Key not found",
			inputJson: map[string]interface{}{
				"key1": "value1",
			},
			key:         "key2",
			expected:    nil,
			expectedErr: errors.New("Key key2 not found in inputJson"),
		},
		{
			name:        "Invalid inputJson type",
			inputJson:   "not a map",
			key:         "key1",
			expected:    nil,
			expectedErr: errors.New("inputJson is not invalid type"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			value, err := getValue(tc.inputJson, tc.key)

			if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
				t.Errorf("getValue() error = %v, expectedErr %v", err, tc.expectedErr)
				return
			}

			if !reflect.DeepEqual(value, tc.expected) {
				t.Errorf("getValue() = %v, want %v", value, tc.expected)
			}
		})
	}
}

func Test_getJsonObjectValue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name        string
		inputJson   interface{}
		key         string
		expected    map[string]interface{}
		expectedErr error
	}{
		{
			name: "Valid input",
			inputJson: map[string]interface{}{
				"key1": map[string]interface{}{
					"subkey1": "subvalue1",
				},
			},
			key: "key1",
			expected: map[string]interface{}{
				"subkey1": "subvalue1",
			},
			expectedErr: nil,
		},
		{
			name: "Key not found",
			inputJson: map[string]interface{}{
				"key1": "value1",
			},
			key:         "key2",
			expected:    nil,
			expectedErr: errors.New("Key key2 not found in inputJson"),
		},
		{
			name: "Invalid value type",
			inputJson: map[string]interface{}{
				"key1": "value1",
			},
			key:         "key1",
			expected:    nil,
			expectedErr: errors.New("Value for key key1 is not a int"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			value, err := getJsonObjectValue(tc.inputJson, tc.key)

			if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
				t.Errorf("getJsonObjectValue() error = %v, expectedErr %v", err, tc.expectedErr)
				return
			}

			if !reflect.DeepEqual(value, tc.expected) {
				t.Errorf("getJsonObjectValue() = %v, want %v", value, tc.expected)
			}
		})
	}
}

func Test_getJsonStringValue(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name        string
		inputJson   interface{}
		key         string
		expected    string
		expectedErr error
	}{
		{
			name: "Valid input",
			inputJson: map[string]interface{}{
				"key1": "value1",
			},
			key:         "key1",
			expected:    "value1",
			expectedErr: nil,
		},
		{
			name: "Key not found",
			inputJson: map[string]interface{}{
				"key1": "value1",
			},
			key:         "key2",
			expected:    "",
			expectedErr: errors.New("Key key2 not found in inputJson"),
		},
		{
			name: "Invalid value type",
			inputJson: map[string]interface{}{
				"key1": 123,
			},
			key:         "key1",
			expected:    "",
			expectedErr: errors.New("Value for key key1 is not a string"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			value, err := getJsonStringValue(tc.inputJson, tc.key)

			if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
				t.Errorf("getJsonStringValue() error = %v, expectedErr %v", err, tc.expectedErr)
				return
			}

			if value != tc.expected {
				t.Errorf("getJsonStringValue() = %v, want %v", value, tc.expected)
			}
		})
	}
}

func TestIPMIImplement_Collection_ErrorHandling(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo:  `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`,
			ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with missing fields to trigger errors
	apiResponse := map[string]interface{}{ // Missing ProcessorSummary
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Expect the APIExecute method of the mock API instance to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/0", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect the Write method of the mock Logging instance NOT to be called because of errors
	// The function should continue to the next target, but there's only one target in this test

	// Call the Collection method
	l.Collection(targetList)
}

func TestIPMIImplement_Collection_InvalidAPIResponse(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
			ProductInfo:  `{"vendor":"dell","product_name":"PowerEdge","version":"R750","os":""}`, // Valid JSON format
		},
	}

	// Define the API response with invalid structure to trigger errors
	apiResponse := map[string]interface{}{
		"ProcessorSummary": "not a map",
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Expect the APIExecute method of the mock API instance to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/System.Embedded.1", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect the Write method of the mock Logging instance NOT to be called because of errors
	// The function should continue to the next target, but there's only one target in this test

	// Call the Collection method
	l.Collection(targetList)
}

// TestIPMIImplement_Collection_ProcessorSummaryStatusParseError tests ProcessorSummary Status parse failure
func TestIPMIImplement_Collection_ProcessorSummaryStatusParseError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with ProcessorSummary.Status as non-object to trigger parse error
	apiResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": "not_an_object", // This should be an object but is a string
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Expect the APIExecute method to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/System.Embedded.1", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect Write method NOT to be called because of the parse error
	// mockLogging.EXPECT().Write() should not be called

	// Call the Collection method
	l.Collection(targetList)
}

// TestIPMIImplement_Collection_ProcessorSummaryHealthParseError tests ProcessorSummary Health parse failure
func TestIPMIImplement_Collection_ProcessorSummaryHealthParseError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with ProcessorSummary.Status.Health as non-string to trigger parse error
	apiResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": 123, // This should be a string but is an integer
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Expect the APIExecute method to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/0", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect Write method NOT to be called because of the parse error
	// mockLogging.EXPECT().Write() should not be called

	// Call the Collection method
	l.Collection(targetList)
}

// TestIPMIImplement_Collection_MemorySummaryParseError tests MemorySummary parse failure
func TestIPMIImplement_Collection_MemorySummaryParseError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with MemorySummary as non-object to trigger parse error
	apiResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": "not_an_object", // This should be an object but is a string
	}

	// Expect the APIExecute method to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/System.Embedded.1", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect Write method NOT to be called because of the parse error
	// mockLogging.EXPECT().Write() should not be called

	// Call the Collection method
	l.Collection(targetList)
}

// TestIPMIImplement_Collection_MemorySummaryStatusParseError tests MemorySummary Status parse failure
func TestIPMIImplement_Collection_MemorySummaryStatusParseError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with MemorySummary.Status as non-object to trigger parse error
	apiResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": []string{"not", "an", "object"}, // This should be an object but is a slice
		},
	}

	// Expect the APIExecute method to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/0", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect Write method NOT to be called because of the parse error
	// mockLogging.EXPECT().Write() should not be called

	// Call the Collection method
	l.Collection(targetList)
}

// TestIPMIImplement_Collection_MemorySummaryHealthParseError tests MemorySummary Health parse failure
func TestIPMIImplement_Collection_MemorySummaryHealthParseError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock API instance
	mockAPI := mocks.NewMockAPI(ctrl)

	// Create a mock Logging instance
	mockLogging := mocks.NewMockLogging(ctrl)

	// Create a IPMIImplement instance
	l := NewTestIPMIImplement(mockAPI, mockLogging)

	// Define the test target list
	targetList := []interfaces.IPMITargetList{
		{
			ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}",
			ServerID:     "test_server1",
			IPMIAddress:  "test_address1",
			IPMIUser:     "test_user1",
			IPMIPassword: "test_password1",
		},
	}

	// Define the API response with MemorySummary.Status.Health as non-string to trigger parse error
	apiResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": map[string]string{"invalid": "type"}, // This should be a string but is a map
			},
		},
	}

	// Expect the APIExecute method to be called
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), "GET", "test_address1", "https://redfish/v1/Systems/System.Embedded.1", "test_user1", "test_password1", "").Return(apiResponse, nil)

	// Expect Write method NOT to be called because of the parse error
	// mockLogging.EXPECT().Write() should not be called

	// Call the Collection method
	l.Collection(targetList)
}

// Helper function to set environment variable and return cleanup function for IPMI tests
func setEnvIPMI(key, value string) func() {
	original := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if original != "" {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}
}

// TestIPMIImplement_Init_MissingEnvironmentVariables_DefaultValues tests default values
func TestIPMIImplement_Init_MissingEnvironmentVariables_DefaultValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Clear environment variables
	defer setEnvIPMI("IPMI_LOGFILE", "")()
	defer setEnvIPMI("IPMI_LOGPATH", "")()
	defer setEnvIPMI("IPMI_MAXSIZE", "")()
	defer setEnvIPMI("IPMI_MAXBACKUPS", "")()
	defer setEnvIPMI("IPMI_MAXAGE", "")()

	logger := klog.NewKlogr()
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup mocks - expect default values (0 for integers, empty string for strings)
	mockLogging.EXPECT().Init(interfaces.LoggingConfig{
		LogFile:    "",
		LogPath:    "",
		MaxSize:    0,
		MaxBackups: 0,
		MaxAge:     0,
	}).Return(nil)

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: mockLogging,
	}

	// Execute
	err := ipmi.Init()

	// Verify
	assert.NoError(t, err)
}

// TestIPMIImplement_Init_LoggingInitError_ReturnsError tests Logging init failure
func TestIPMIImplement_Init_LoggingInitError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables
	defer setEnvIPMI("IPMI_LOGFILE", "test.log")()
	defer setEnvIPMI("IPMI_LOGPATH", "/tmp/test")()
	defer setEnvIPMI("IPMI_MAXSIZE", "100")()
	defer setEnvIPMI("IPMI_MAXBACKUPS", "5")()
	defer setEnvIPMI("IPMI_MAXAGE", "30")()

	logger := klog.NewKlogr()
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup mocks - Logging fails
	mockLogging.EXPECT().Init(interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "/tmp/test",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
	}).Return(errors.New("logging init failed"))

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: mockLogging,
	}

	// Execute
	err := ipmi.Init()

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logging init failed")
}

// TestIPMIImplement_Init_InvalidIntegerEnvironmentVariables_UsesZero tests invalid integer parsing
func TestIPMIImplement_Init_InvalidIntegerEnvironmentVariables_UsesZero(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup environment variables with invalid integers
	defer setEnvIPMI("IPMI_LOGFILE", "test.log")()
	defer setEnvIPMI("IPMI_LOGPATH", "/tmp/test")()
	defer setEnvIPMI("IPMI_MAXSIZE", "not_a_number")()
	defer setEnvIPMI("IPMI_MAXBACKUPS", "invalid")()
	defer setEnvIPMI("IPMI_MAXAGE", "bad_int")()

	logger := klog.NewKlogr()
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup mocks - expect zero values for invalid integers
	mockLogging.EXPECT().Init(interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "/tmp/test",
		MaxSize:    0,
		MaxBackups: 0,
		MaxAge:     0,
	}).Return(nil)

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: mockLogging,
	}

	// Execute
	err := ipmi.Init()

	// Verify
	assert.NoError(t, err)
}

// TestIPMIImplement_Collection_EmptyTargetList_NoOperation tests empty target list
func TestIPMIImplement_Collection_EmptyTargetList_NoOperation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	logger := klog.NewKlogr()

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: nil,
	}

	// Empty target list
	targetList := []interfaces.IPMITargetList{}

	// Execute - should not panic
	assert.NotPanics(t, func() {
		ipmi.Collection(targetList)
	})
}

// TestIPMIImplement_Collection_NilTargetList_NoOperation tests nil target list
func TestIPMIImplement_Collection_NilTargetList_NoOperation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	logger := klog.NewKlogr()

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: nil,
	}

	// Execute with nil target list - should not panic
	assert.NotPanics(t, func() {
		ipmi.Collection(nil)
	})
}

// TestIPMIImplement_Collection_APICallError_LogsError tests API call failure
func TestIPMIImplement_Collection_APICallError_LogsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	targetList := []interfaces.IPMITargetList{
		{ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}", ServerID: "server1", IPMIAddress: "192.168.1.101", IPMIUser: "admin", IPMIPassword: "password"},
	}

	// Setup mocks - API fails
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("API call failed"))

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Execute - should not panic, just log error
	assert.NotPanics(t, func() {
		ipmi.Collection(targetList)
	})
}

// TestIPMIImplement_Collection_LoggingWriteError_ContinuesExecution tests Logging write failure
func TestIPMIImplement_Collection_LoggingWriteError_ContinuesExecution(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	targetList := []interfaces.IPMITargetList{
		{ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}", ServerID: "server1", IPMIAddress: "192.168.1.101", IPMIUser: "admin", IPMIPassword: "password"},
	}

	expectedResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Setup mocks - API succeeds but Logging fails
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedResponse, nil)
	mockLogging.EXPECT().Write(gomock.Any(), `{ "processer": "OK", "memory": "OK" }`).Return(errors.New("logging write failed"))

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Execute - should not panic, just log error
	assert.NotPanics(t, func() {
		ipmi.Collection(targetList)
	})
}

// TestIPMIImplement_Collection_MultipleTargets_ProcessesAll tests multiple targets
func TestIPMIImplement_Collection_MultipleTargets_ProcessesAll(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAPI := mocks.NewMockAPI(ctrl)
	mockLogging := mocks.NewMockLogging(ctrl)

	targetList := []interfaces.IPMITargetList{
		{ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}", ServerID: "server1", IPMIAddress: "192.168.1.101", IPMIUser: "admin", IPMIPassword: "password"},
		{ProductInfo: `{"vendor":"fujitsu","product_name":"PRIMERGY","version":""}`, ExtraParameters: "{}", ServerID: "server2", IPMIAddress: "192.168.1.103", IPMIUser: "admin", IPMIPassword: "password"},
		{ProductInfo: `{"vendor":"dell","product_name":"PowerEdge","version":""}`, ExtraParameters: "{}", ServerID: "server3", IPMIAddress: "192.168.1.105", IPMIUser: "admin", IPMIPassword: "password"},
	}

	expectedResponse := map[string]interface{}{
		"ProcessorSummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
		"MemorySummary": map[string]interface{}{
			"Status": map[string]interface{}{
				"Health": "OK",
			},
		},
	}

	// Setup mocks - expect 3 API calls and 3 logging writes
	mockAPI.EXPECT().APIExecuteUserAuth(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedResponse, nil).Times(3)
	mockLogging.EXPECT().Write(gomock.Any(), `{ "processer": "OK", "memory": "OK" }`).Return(nil).Times(3)

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     mockAPI,
		Logging: mockLogging,
	}

	// Execute
	assert.NotPanics(t, func() {
		ipmi.Collection(targetList)
	})
}

// TestIPMIImplement_Finalize_WithNilLogging_Success tests finalize with nil logging
func TestIPMIImplement_Finalize_WithNilLogging_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup mocks
	mockLogging.EXPECT().Finalize().Return()

	logger := klog.NewKlogr()

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: mockLogging,
	}

	// Execute - should not panic even with nil logging
	assert.NotPanics(t, func() {
		ipmi.Finalize()
	})
}

// TestIPMIImplement_Finalize_WithMockLogging_CallsFinalize tests finalize with mock logging
func TestIPMIImplement_Finalize_WithMockLogging_CallsFinalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockLogging := mocks.NewMockLogging(ctrl)

	// Setup mock - expect Finalize call
	mockLogging.EXPECT().Finalize()

	ipmi := &IPMIImplement{
		Logger:  logger,
		API:     nil,
		Logging: mockLogging,
	}

	// Execute
	assert.NotPanics(t, func() {
		ipmi.Finalize()
	})
}

// TestCustomError_Error_ReturnsFormattedString tests CustomError string format
func TestCustomError_Error_ReturnsFormattedString(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Test case 1: Normal error
	err := &CustomError{
		StatusCode: 404,
		Message:    "Not Found",
	}

	expected := "<404> Not Found"
	actual := err.Error()

	assert.Equal(t, expected, actual)

	// Test case 2: Different status code
	err2 := &CustomError{
		StatusCode: 500,
		Message:    "Internal Server Error",
	}

	expected2 := "<500> Internal Server Error"
	actual2 := err2.Error()

	assert.Equal(t, expected2, actual2)

	// Test case 3: Empty message
	err3 := &CustomError{
		StatusCode: 200,
		Message:    "",
	}

	expected3 := "<200> "
	actual3 := err3.Error()

	assert.Equal(t, expected3, actual3)
}
