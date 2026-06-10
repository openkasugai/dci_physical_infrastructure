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
	"context"
	"errors"
	"log_module/internal/server/interfaces"
	"log_module/internal/server/test_utils"
	"os"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

// MockAnsible is a mock implementation of the Ansible interface.
type MockAnsible struct {
	Ctrl     *gomock.Controller
	recorder *MockAnsibleMockRecorder
}

// MockAnsibleMockRecorder is the mock recorder for MockAnsible.
type MockAnsibleMockRecorder struct {
	mock *MockAnsible
}

// NewMockAnsible creates a new mock instance.
func NewMockAnsible(ctrl *gomock.Controller) *MockAnsible {
	mock := &MockAnsible{Ctrl: ctrl}
	mock.recorder = &MockAnsibleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAnsible) EXPECT() *MockAnsibleMockRecorder {
	return m.recorder
}

// CmdExecute mocks base method.
func (m *MockAnsible) CmdExecute(ctx context.Context, host, user, playbook, extraArgs string) (interface{}, error) {
	m.Ctrl.T.Helper()
	ret := m.Ctrl.Call(m, "CmdExecute", ctx, host, user, playbook, extraArgs)
	ret0 := ret[0]
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CmdExecute indicates an expected call of CmdExecute.
func (mr *MockAnsibleMockRecorder) CmdExecute(ctx, host, user, playbook, extraArgs interface{}) *gomock.Call {
	mr.mock.Ctrl.T.Helper()
	return mr.mock.Ctrl.RecordCallWithMethodType(mr.mock, "CmdExecute", reflect.TypeOf((*MockAnsible)(nil).CmdExecute), ctx, host, user, playbook, extraArgs)
}

// MockLogging is a mock implementation of the Logging interface.
type MockLogging struct {
	Ctrl     *gomock.Controller
	recorder *MockLoggingMockRecorder
}

// MockLoggingMockRecorder is the mock recorder for MockLogging.
type MockLoggingMockRecorder struct {
	mock *MockLogging
}

// NewMockLogging creates a new mock instance.
func NewMockLogging(ctrl *gomock.Controller) *MockLogging {
	mock := &MockLogging{Ctrl: ctrl}
	mock.recorder = &MockLoggingMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLogging) EXPECT() *MockLoggingMockRecorder {
	return m.recorder
}

// Init mocks base method.
func (m *MockLogging) Init(config interfaces.LoggingConfig) error {
	m.Ctrl.T.Helper()
	ret := m.Ctrl.Call(m, "Init", config)
	ret0, _ := ret[0].(error)
	return ret0
}

// Init indicates an expected call of Init.
func (mr *MockLoggingMockRecorder) Init(config interface{}) *gomock.Call {
	mr.mock.Ctrl.T.Helper()
	return mr.mock.Ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockLogging)(nil).Init), config)
}

// Finalize mocks base method.
func (m *MockLogging) Finalize() {
	m.Ctrl.T.Helper()
	m.Ctrl.Call(m, "Finalize")
}

// Finalize indicates an expected call of Finalize.
func (mr *MockLoggingMockRecorder) Finalize() *gomock.Call {
	mr.mock.Ctrl.T.Helper()
	return mr.mock.Ctrl.RecordCallWithMethodType(mr.mock, "Finalize", reflect.TypeOf((*MockLogging)(nil).Finalize))
}

// Write mocks base method.
func (m *MockLogging) Write(keyId, json string) error {
	m.Ctrl.T.Helper()
	ret := m.Ctrl.Call(m, "Write", keyId, json)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockLoggingMockRecorder) Write(keyId, json interface{}) *gomock.Call {
	mr.mock.Ctrl.T.Helper()
	return mr.mock.Ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockLogging)(nil).Write), keyId, json)
}

func NewTestCDIImplement(ansible interfaces.Ansible, logging interfaces.Logging) *CDIImplement {
	logger := klog.Background() // Get a klog.Logger instance
	return &CDIImplement{
		Logger:  logger,
		Ansible: ansible,
		Logging: logging,
	}
}

func TestCDIImplement_Init(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Logging instance
	mockLogging := NewMockLogging(ctrl)

	// Create a CDIImplement instance
	l := CDIImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Set necessary environment variables for the test
	_ = os.Setenv("CDI_LOGFILE", "test_cdi.log")
	_ = os.Setenv("CDI_LOGPATH", "/tmp")
	_ = os.Setenv("CDI_MAXSIZE", "1024")
	_ = os.Setenv("CDI_MAXBACKUPS", "5")
	_ = os.Setenv("CDI_MAXAGE", "7")

	// Expect the Init method of the mock Logging instance to be called
	mockLogging.EXPECT().Init(gomock.Any()).Return(nil)

	// Call the Init method
	err := l.Init()

	// Assert that there is no error
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Clean up the environment variables
	_ = os.Unsetenv("CDI_LOGFILE")
	_ = os.Unsetenv("CDI_LOGPATH")
	_ = os.Unsetenv("CDI_MAXSIZE")
	_ = os.Unsetenv("CDI_MAXBACKUPS")
	_ = os.Unsetenv("CDI_MAXAGE")
}

func TestCDIImplement_Finalize(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Logging instance
	mockLogging := NewMockLogging(ctrl)

	// Create a CDIImplement instance
	l := CDIImplement{
		Logger:  klog.Background(),
		Logging: mockLogging,
	}

	// Expect the Finalize method of the mock Logging instance to be called
	mockLogging.EXPECT().Finalize()

	// Call the Finalize method
	l.Finalize()
}

func TestCDIImplement_Collection(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Ansible instance
	mockAnsible := NewMockAnsible(ctrl)

	// Create a mock Logging instance
	mockLogging := NewMockLogging(ctrl)

	// Create a CDIImplement instance
	l := NewTestCDIImplement(mockAnsible, mockLogging)

	// Set necessary environment variables for the test
	_ = os.Setenv("SSH_KEY", "test_ssh_key")

	// Define the test target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"test_guest1","cdimgr_guest_user":"test_user1","cdimgr_guest_password":"test_guest_password1","cdimgr_host_password":"test_host_password1","director_password":"test_director_password1"}`,
		},
		{
			CDIHost:         "192.168.1.2",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"test_guest2","cdimgr_guest_user":"test_user2","cdimgr_guest_password":"test_guest_password2","cdimgr_host_password":"test_host_password2","director_password":"test_director_password2"}`,
		},
	}

	// Define the expected Ansible command output
	ansibleOutput := []interface{}{
		"category1 key1=value1",
		"category2 key2=value2",
	}

	// Expect the CmdExecute method of the mock Ansible instance to be called for each target
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "test_guest1", "test_user1", "cdi_hh.yaml", "cdi_pass=test_host_password1 director_pass=test_director_password1").Return(ansibleOutput, nil)
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "test_guest2", "test_user2", "cdi_hh.yaml", "cdi_pass=test_host_password2 director_pass=test_director_password2").Return(ansibleOutput, nil)

	// Expect the Write method of the mock Logging instance to be called for each target
	mockLogging.EXPECT().Write("test_guest1", gomock.Any()).Return(nil)
	mockLogging.EXPECT().Write("test_guest2", gomock.Any()).Return(nil)

	// Call the Collection method
	l.Collection(targetList)

	// Clean up the environment variables
	_ = os.Unsetenv("SSH_KEY")
}

func TestCDIImplement_parseMsg(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	testCases := []struct {
		name        string
		input       []interface{}
		expected    string
		expectedErr error
	}{
		{
			name: "Valid input",
			input: []interface{}{
				"category1 key1=value1",
				"category2 key2=value2",
			},
			expected: `
{
  "category1": {
    "key1": "value1"
  },
  "category2": {
    "key2": "value2"
  }
}
`,
			expectedErr: nil,
		},
		{
			name: "Invalid input - missing =",
			input: []interface{}{
				"category1 key1 value1",
			},
			expected:    `{}`,
			expectedErr: nil,
		},
		{
			name: "Invalid input - missing space in key",
			input: []interface{}{
				"category1key1=value1",
			},
			expected:    `{}`,
			expectedErr: nil,
		},
	}

	l := &CDIImplement{Logger: klog.Background()}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
			defer cleanup()

			jsonOut, err := l.parseMsg(tc.input)

			if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
				t.Errorf("parseMsg() error = %v, expectedErr %v", err, tc.expectedErr)
				return
			}

			expected := strings.TrimSpace(tc.expected)
			jsonOut = strings.TrimSpace(jsonOut)

			if jsonOut != expected {
				t.Errorf("parseMsg() = %v, want %v", jsonOut, expected)
			}
		})
	}
}

func TestCDIImplement_Collection_CmdExecuteError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	// Create a mock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock Ansible instance
	mockAnsible := NewMockAnsible(ctrl)

	// Create a mock Logging instance
	mockLogging := NewMockLogging(ctrl)

	// Create a CDIImplement instance
	l := NewTestCDIImplement(mockAnsible, mockLogging)

	// Set necessary environment variables for the test
	_ = os.Setenv("SSH_KEY", "test_ssh_key")

	// Define the test target list
	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"test_guest1","cdimgr_guest_user":"test_user1","cdimgr_guest_password":"test_guest_password1","cdimgr_host_password":"test_host_password1","director_password":"test_director_password1"}`,
		},
	}

	// Expect the CmdExecute method of the mock Ansible instance to be called and return an error
	expectedError := errors.New("cmd execute error")
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "test_guest1", "test_user1", "cdi_hh.yaml", "cdi_pass=test_host_password1 director_pass=test_director_password1").Return(nil, expectedError)

	// Call the Collection method
	l.Collection(targetList)

	// Clean up the environment variables
	_ = os.Unsetenv("SSH_KEY")
}

// Helper function to set environment variable and return cleanup function for CDI tests
func setEnvCDI(key, value string) func() {
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

// TestCDIImplement_Init_MissingEnvironmentVariables_DefaultValues tests default values
func TestCDIImplement_Init_MissingEnvironmentVariables_DefaultValues(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Clear environment variables
	defer setEnvCDI("CDI_LOGFILE", "")()
	defer setEnvCDI("CDI_LOGPATH", "")()
	defer setEnvCDI("CDI_MAXSIZE", "")()
	defer setEnvCDI("CDI_MAXBACKUPS", "")()
	defer setEnvCDI("CDI_MAXAGE", "")()

	logger := klog.NewKlogr()
	mockLogging := NewMockLogging(ctrl)

	// Setup mocks - expect default values (0 for integers, empty string for strings)
	mockLogging.EXPECT().Init(interfaces.LoggingConfig{
		LogFile:    "",
		LogPath:    "",
		MaxSize:    0,
		MaxBackups: 0,
		MaxAge:     0,
	}).Return(nil)

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: nil,
		Logging: mockLogging,
	}

	// Execute
	err := cdi.Init()

	// Verify
	assert.NoError(t, err)
}

// TestCDIImplement_ParseMsg_ValidInput_ReturnsJson tests parseMsg method
func TestCDIImplement_ParseMsg_ValidInput_ReturnsJson(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	cdi := &CDIImplement{Logger: logger}

	// Test input
	input := []interface{}{
		"category1 key1=value1",
		"category1 key2=value2",
		"category2 key1=value3",
	}

	// Execute
	result, err := cdi.parseMsg(input)

	// Verify
	assert.NoError(t, err)
	assert.Contains(t, result, "category1")
	assert.Contains(t, result, "category2")
	assert.Contains(t, result, "key1")
	assert.Contains(t, result, "value1")
}

// TestCDIImplement_ParseMsg_InvalidInput_ReturnsError tests invalid input type
func TestCDIImplement_ParseMsg_InvalidInput_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	cdi := &CDIImplement{Logger: logger}

	// Test with invalid input type
	input := "not an array"

	// Execute
	result, err := cdi.parseMsg(input)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input is not []interface{}")
	assert.Empty(t, result)
}

// TestCDIImplement_ParseMsg_NonStringElements_SkipsElements tests non-string elements
func TestCDIImplement_ParseMsg_NonStringElements_SkipsElements(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	cdi := &CDIImplement{Logger: logger}

	// Test input with mixed types
	input := []interface{}{
		"category1 key1=value1",
		123, // non-string element
		"category1 key2=value2",
		nil, // nil element - this will be skipped
	}

	// Execute
	result, err := cdi.parseMsg(input)

	// Verify
	assert.NoError(t, err)
	assert.Contains(t, result, "category1")
	assert.Contains(t, result, "key1")
	assert.Contains(t, result, "key2")
}

// TestCDIImplement_ParseMsg_EmptyInput_ReturnsEmptyJson tests empty input
func TestCDIImplement_ParseMsg_EmptyInput_ReturnsEmptyJson(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	logger := klog.NewKlogr()
	cdi := &CDIImplement{Logger: logger}

	// Test with empty input
	input := []interface{}{}

	// Execute
	result, err := cdi.parseMsg(input)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "{}", result)
}

// TestCDIImplement_Collection_EmptyTargetList_NoOperation tests empty target list
func TestCDIImplement_Collection_EmptyTargetList_NoOperation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	logger := klog.NewKlogr()

	// Set up SSH_KEY environment variable
	defer setEnvCDI("SSH_KEY", "test_key")()

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: nil,
	}

	// Empty target list
	targetList := []interfaces.CDITargetList{}

	// Execute - should not panic
	assert.NotPanics(t, func() {
		cdi.Collection(targetList)
	})
}

// TestCDIImplement_Collection_NilTargetList_NoOperation tests nil target list
func TestCDIImplement_Collection_NilTargetList_NoOperation(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	logger := klog.NewKlogr()

	// Set up SSH_KEY environment variable
	defer setEnvCDI("SSH_KEY", "test_key")()

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: nil,
	}

	// Execute with nil target list - should not panic
	assert.NotPanics(t, func() {
		cdi.Collection(nil)
	})
}

// TestCDIImplement_Finalize_WithNilDependencies_Success tests finalize with nil dependencies
func TestCDIImplement_Finalize_WithNilDependencies_Success(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogging := NewMockLogging(ctrl)

	// Setup mocks
	mockLogging.EXPECT().Finalize().Return()

	logger := klog.NewKlogr()

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: nil,
		Logging: mockLogging,
	}

	// Execute - should not panic even with nil dependencies
	assert.NotPanics(t, func() {
		cdi.Finalize()
	})
}

// TestCDIImplement_Init_LoggingInitError_ReturnsError tests Init when Logging.Init fails
func TestCDIImplement_Init_LoggingInitError_ReturnsError(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Clear environment variables
	defer setEnvCDI("CDI_LOGFILE", "test.log")()
	defer setEnvCDI("CDI_LOGPATH", "/tmp")()
	defer setEnvCDI("CDI_MAXSIZE", "100")()
	defer setEnvCDI("CDI_MAXBACKUPS", "5")()
	defer setEnvCDI("CDI_MAXAGE", "30")()

	logger := klog.NewKlogr()
	mockLogging := NewMockLogging(ctrl)

	// Setup mocks - Logging.Init fails
	mockLogging.EXPECT().Init(interfaces.LoggingConfig{
		LogFile:    "test.log",
		LogPath:    "/tmp",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
	}).Return(errors.New("logging init failed"))

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: nil,
		Logging: mockLogging,
	}

	// Execute
	err := cdi.Init()

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logging init failed")
}

// TestCDIImplement_Collection_ParseMsgError_ReturnsEarly tests parseMsg failure
func TestCDIImplement_Collection_ParseMsgError_ReturnsEarly(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAnsible := NewMockAnsible(ctrl)
	mockLogging := NewMockLogging(ctrl)

	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest1","cdimgr_guest_password":"guestPass1","cdimgr_host_password":"hostPass1","director_password":"directorPass1"}`,
		},
	}

	// Setup mocks - Ansible succeeds but returns invalid output for parseMsg
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.10.1", "guest1", "cdi_hh.yaml", "cdi_pass=hostPass1 director_pass=directorPass1").
		Return("not an array", nil) // This will cause parseMsg to fail

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Execute - should return early due to parseMsg error
	assert.NotPanics(t, func() {
		cdi.Collection(targetList)
	})
}

// TestCDIImplement_Collection_LoggingWriteError_ContinuesWithNext tests Logging.Write failure
func TestCDIImplement_Collection_LoggingWriteError_ContinuesWithNext(t *testing.T) {
	cleanup := test_utils.SetupTestEnvironmentWithKlog(t)
	defer cleanup()
cleanupMappings := test_utils.SetupProductMappings()
defer cleanupMappings()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := klog.NewKlogr()
	mockAnsible := NewMockAnsible(ctrl)
	mockLogging := NewMockLogging(ctrl)

	targetList := []interfaces.CDITargetList{
		{
			CDIHost:         "192.168.1.1",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"192.168.10.1","cdimgr_guest_user":"guest1","cdimgr_guest_password":"guestPass1","cdimgr_host_password":"hostPass1","director_password":"directorPass1"}`,
		},
		{
			CDIHost:         "192.168.1.2",
			ProductInfo:     `{"vendor":"fujitsu","product_name":"PRIMERGY CDI","version":"1.1"}`,
			ExtraParameters: `{"cdi_guest":"192.168.10.2","cdimgr_guest_user":"guest2","cdimgr_guest_password":"guestPass2","cdimgr_host_password":"hostPass2","director_password":"directorPass2"}`,
		},
	}

	// Valid output for parseMsg
	validOutput := []interface{}{
		"category1 key1=value1",
		"category1 key2=value2",
	}

	// Setup mocks - first target fails on Logging.Write, second succeeds
	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.10.1", "guest1", "cdi_hh.yaml", "cdi_pass=hostPass1 director_pass=directorPass1").
		Return(validOutput, nil)
	mockLogging.EXPECT().Write("192.168.10.1", gomock.Any()).Return(errors.New("logging write failed"))

	mockAnsible.EXPECT().CmdExecute(gomock.Any(), "192.168.10.2", "guest2", "cdi_hh.yaml", "cdi_pass=hostPass2 director_pass=directorPass2").
		Return(validOutput, nil)
	mockLogging.EXPECT().Write("192.168.10.2", gomock.Any()).Return(nil)

	cdi := &CDIImplement{
		Logger:  logger,
		Ansible: mockAnsible,
		Logging: mockLogging,
	}

	// Execute - should continue processing despite first target failing
	assert.NotPanics(t, func() {
		cdi.Collection(targetList)
	})
}
