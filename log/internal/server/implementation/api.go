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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

	"k8s.io/klog/v2"
)

// struct of API
type APIImplement struct {
	Logger klog.Logger
}

// API execution with user authentication
func (l APIImplement) APIExecuteUserAuth(ctx context.Context, method string, url string, apiname string, loginUser string, loginPass string, queryParameter string) (jsonData interface{}, err error) {
	l.Logger.V(2).Info("start APIExecuteUserAuth", "method", method, "url", url, "apiname", apiname, "loginUser", loginUser)
	defer func() {
		l.Logger.V(2).Info("end APIExecuteUserAuth", "jsonData", jsonData, "err", err)
	}()

	l.Logger.Info("executing API request with user authentication", "url", url, "api", apiname, "method", method)

	// execute http request.
	resp, e := l.execRequestUserAuth(method, url, apiname, loginUser, loginPass, queryParameter)

	// error check.
	if e != nil {
		l.Logger.V(2).Info("branch: execRequestUserAuth failed", "error", e.Error())
		err = e
		return
	}

	l.Logger.V(2).Info("branch: execRequestUserAuth succeeded")
	jsonData = resp
	err = nil
	return
}

// API execution with JWT authentication
func (l APIImplement) APIExecuteJWTAUth(ctx context.Context, method string, url string, apiname string, jwt string, queryParameter string) (jsonData interface{}, err error) {
	l.Logger.V(2).Info("start APIExecuteJWTAUth", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(2).Info("end APIExecuteJWTAUth", "jsonData", jsonData, "err", err)
	}()
	
	l.Logger.Info("executing API request with JWT", "url", url, "api", apiname, "method", method)
	// execute http request.
	resp, e := l.execRequestJWTAuth(method, url, apiname, jwt, queryParameter)

	// error check.
	if e != nil {
		l.Logger.V(2).Info("branch: execRequestJWTAuth failed", "error", e.Error())
		err = e
		return
	}
	
	l.Logger.V(2).Info("branch: execRequestJWTAuth succeeded")
	jsonData = resp
	err = nil
	return
}

func (l APIImplement) execRequestUserAuth(method string, url string, apiname string, loginUser string, loginPass string, queryParameter string) (resp interface{}, err error) {
	l.Logger.V(2).Info("start execRequest", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(2).Info("end execRequest", "err", err)
	}()

	// generate URL
	reqUrl := fmt.Sprintf(`%s/%s`, url, apiname)
	if queryParameter != "" {
		reqUrl = fmt.Sprintf(`%s?%s`, reqUrl, queryParameter)
	}
	l.Logger.V(2).Info("generated request URL", "url", reqUrl)

	// create request.
	var reqNr *http.Request
	reqNr, err = http.NewRequest(method, reqUrl, nil)
	if err != nil {
		l.Logger.V(2).Info("branch: NewRequest failed", "error", err.Error())
		return
	}
	reqNr.SetBasicAuth(loginUser, loginPass)
	reqNr.Header.Set("Accept", "application/json")

	l.Logger.V(2).Info("branch: HTTP request created successfully")

	// send request.
	resp, err = l.execRequest(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: execRequest failed", "error", err.Error())
		return
	}
	
	l.Logger.V(2).Info("branch: execRequest succeeded")
	return
}

func (l APIImplement) execRequestJWTAuth(method string, url string, apiname string, jwt string, queryParameter string) (resp interface{}, err error) {
	l.Logger.V(2).Info("start execRequestJWTAuth", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(2).Info("end execRequestJWTAuth", "err", err)
	}()

	// generate URL
	reqUrl := fmt.Sprintf(`%s/%s`, url, apiname)
	if queryParameter != "" {
		reqUrl = fmt.Sprintf(`%s?%s`, reqUrl, queryParameter)
	}
	l.Logger.V(2).Info("generated request URL", "url", reqUrl)

	// create request.
	var reqNr *http.Request
	reqNr, err = http.NewRequest(method, reqUrl, nil)
	if err != nil {
		l.Logger.V(2).Info("branch: NewRequest failed", "error", err.Error())
		return
	}
	reqNr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	reqNr.Header.Set("Accept", "application/json")

	l.Logger.V(2).Info("branch: HTTP request created successfully")

	// send request.
	resp, err = l.execRequest(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: execRequest failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: execRequest succeeded")
	return
}

func (l APIImplement) execRequest(reqNr *http.Request) (resp interface{}, err error) {
	l.Logger.V(2).Info("start execRequest")
	defer func() {
		l.Logger.V(2).Info("end execRequest", "err", err)
	}()

	// logging for debug
	dump, err := httputil.DumpRequestOut(reqNr, true)
	if err == nil {
		l.Logger.V(2).Info("HTTP request dump", "dump", string(dump))
	} else {
		l.Logger.V(2).Info("branch: HTTP request dump failed", "error", err.Error())
	}

	// tls check disable
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			PreferServerCipherSuites: true,
			InsecureSkipVerify:       true,
			// MinVersion:               tls.VersionTLS12,
			// MaxVersion:               tls.VersionTLS12,
			CipherSuites: []uint16{
				// TLS 1.0 - 1.2 cipher suites.
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				// TLS 1.3 cipher suites.
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
	}
	client := &http.Client{Transport: tr}
	l.Logger.V(2).Info("branch: HTTP client configured with TLS settings")

	// send request.
	respCd, err := client.Do(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: HTTP request failed", "error", err.Error())
		return
	}
	defer func() { _ = respCd.Body.Close() }()

	l.Logger.V(2).Info("branch: HTTP request completed", "statusCode", respCd.StatusCode)

	// read response body.
	result, err := io.ReadAll(respCd.Body)
	if err != nil {
		l.Logger.V(2).Info("branch: response body read failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: response body read successfully", "bodySize", len(result))

	// HTTP status check
	if respCd.StatusCode < 200 || respCd.StatusCode >= 300 { // not 2xx -> error status
		l.Logger.V(2).Info("branch: HTTP status error", "statusCode", respCd.StatusCode, "response", string(result))
		err = &CustomError{StatusCode: respCd.StatusCode, Message: string(result)}
		return
	}

	l.Logger.V(2).Info("branch: HTTP status successful", "statusCode", respCd.StatusCode)

	// extract JSON response
	resp, err = l.extracJsonResponse(result)
	if err != nil {
		l.Logger.V(2).Info("branch: JSON extraction failed", "error", err.Error())
		return
	}

	l.Logger.V(2).Info("branch: JSON extraction successful")
	return
}

func (l APIImplement) extracJsonResponse(ressponseBody []byte) (jsonResponse interface{}, err error) {
	l.Logger.V(2).Info("start extracJsonResponse", "bodySize", len(ressponseBody))
	defer func() {
		l.Logger.V(2).Info("end extracJsonResponse", "err", err)
	}()

	var result interface{}
	err = json.Unmarshal(ressponseBody, &result)
	if err != nil {
		l.Logger.V(2).Info("branch: JSON unmarshal failed", "error", err.Error())
		err = errors.New("response body is invalid for json format")
		return
	}

	l.Logger.V(2).Info("branch: JSON unmarshal successful")
	jsonResponse = result
	return
}
