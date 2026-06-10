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

package mocks

import (
	"io"
	"net/http"
	"strings"
)

// MockHTTPClient for testing
type MockHTTPClient struct {
	MockResponse *http.Response
	MockError    error
	RequestLog   []*http.Request
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Log the request for verification
	m.RequestLog = append(m.RequestLog, req)

	if m.MockError != nil {
		return nil, m.MockError
	}

	return m.MockResponse, nil
}

// Helper function to create mock response
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Helper function to create mock response with custom headers
func NewMockResponseWithHeaders(statusCode int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

// GetLastRequest returns the last captured request
func (m *MockHTTPClient) GetLastRequest() *http.Request {
	if len(m.RequestLog) == 0 {
		return nil
	}
	return m.RequestLog[len(m.RequestLog)-1]
}

// GetRequestCount returns the number of requests made
func (m *MockHTTPClient) GetRequestCount() int {
	return len(m.RequestLog)
}

// Reset clears the request log
func (m *MockHTTPClient) Reset() {
	m.RequestLog = nil
	m.MockResponse = nil
	m.MockError = nil
}
