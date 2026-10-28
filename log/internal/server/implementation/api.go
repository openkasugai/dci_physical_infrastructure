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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

// struct of API
type APIImplement struct {
	Logger klog.Logger
}

// API execution with user authentication
func (l APIImplement) APIExecuteUserAuth(ctx context.Context, method string, url string, apiname string, loginUser string, loginPass string, queryParameter string) (jsonData interface{}, err error) {
	l.Logger.V(3).Info("start APIExecuteUserAuth", "method", method, "url", url, "apiname", apiname, "loginUser", loginUser)
	defer func() {
		l.Logger.V(3).Info("end APIExecuteUserAuth", "err", err)
		l.Logger.V(3).Info("APIExecuteUserAuth response detail", "jsonData", jsonData)
	}()

	l.Logger.V(3).Info("executing API request with user authentication", "url", url, "api", apiname, "method", method)

	// execute http request.
	resp, e := l.execRequestUserAuth(method, url, apiname, loginUser, loginPass, queryParameter)

	// error check.
	if e != nil {
		l.Logger.V(2).Info("branch: execRequestUserAuth failed", "error", e.Error())
		err = e
		return
	}

	l.Logger.V(3).Info("branch: execRequestUserAuth succeeded")
	jsonData = resp
	err = nil
	return
}

// API execution with JWT authentication
func (l APIImplement) APIExecuteJWTAUth(ctx context.Context, method string, url string, apiname string, jwt string, queryParameter string) (jsonData interface{}, err error) {
	l.Logger.V(3).Info("start APIExecuteJWTAUth", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(3).Info("end APIExecuteJWTAUth", "err", err)
		l.Logger.V(3).Info("APIExecuteJWTAUth response detail", "jsonData", jsonData)
	}()

	l.Logger.V(3).Info("executing API request with JWT", "url", url, "api", apiname, "method", method)
	// execute http request.
	resp, e := l.execRequestJWTAuth(method, url, apiname, jwt, queryParameter)

	// error check.
	if e != nil {
		l.Logger.V(2).Info("branch: execRequestJWTAuth failed", "error", e.Error())
		err = e
		return
	}

	l.Logger.V(3).Info("branch: execRequestJWTAuth succeeded")
	jsonData = resp
	err = nil
	return
}

// API execution with OAuth authentication
func (l APIImplement) APIExecuteOAuth(ctx context.Context, method string, url string, apiname string, apiKey string, queryParameter string) (jsonData interface{}, err error) {
	l.Logger.V(3).Info("start APIExecuteOAuth", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(3).Info("end APIExecuteOAuth", "err", err)
		l.Logger.V(3).Info("APIExecuteOAuth response detail", "jsonData", jsonData)
	}()

	l.Logger.V(3).Info("executing API request with JWT", "url", url, "api", apiname, "method", method)
	// execute http request.
	resp, e := l.execRequestOAuth(method, url, apiname, apiKey, queryParameter)

	// error check.
	if e != nil {
		l.Logger.V(2).Info("branch: APIExecuteOAuth failed", "error", e.Error())
		err = e
		return
	}

	l.Logger.V(3).Info("branch: APIExecuteOAuth succeeded")
	jsonData = resp
	err = nil
	return
}

func (l APIImplement) execRequestUserAuth(method string, url string, apiname string, loginUser string, loginPass string, queryParameter string) (resp interface{}, err error) {
	l.Logger.V(3).Info("start execRequest", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(3).Info("end execRequest", "err", err)
	}()

	// generate URL
	reqUrl := fmt.Sprintf(`%s/%s`, url, apiname)
	if queryParameter != "" {
		reqUrl = fmt.Sprintf(`%s?%s`, reqUrl, queryParameter)
	}
	l.Logger.V(3).Info("generated request URL", "url", reqUrl)

	// create request.
	var reqNr *http.Request
	reqNr, err = http.NewRequest(method, reqUrl, nil)
	if err != nil {
		l.Logger.V(2).Info("branch: NewRequest failed", "error", err.Error())
		return
	}
	reqNr.SetBasicAuth(loginUser, loginPass)
	reqNr.Header.Set("Accept", "application/json")

	l.Logger.V(3).Info("branch: HTTP request created successfully")

	// send request.
	resp, err = l.execRequest(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: execRequest failed", "error", err.Error())
		return
	}

	l.Logger.V(3).Info("branch: execRequest succeeded")
	return
}

func (l APIImplement) execRequestJWTAuth(method string, url string, apiname string, jwt string, queryParameter string) (resp interface{}, err error) {
	l.Logger.V(3).Info("start execRequestJWTAuth", "method", method, "url", url, "apiname", apiname)
	defer func() {
		l.Logger.V(3).Info("end execRequestJWTAuth", "err", err)
	}()

	// generate URL
	reqUrl := fmt.Sprintf(`%s/%s`, url, apiname)
	if queryParameter != "" {
		reqUrl = fmt.Sprintf(`%s?%s`, reqUrl, queryParameter)
	}
	l.Logger.V(3).Info("generated request URL", "url", reqUrl)

	// create request.
	var reqNr *http.Request
	reqNr, err = http.NewRequest(method, reqUrl, nil)
	if err != nil {
		l.Logger.V(2).Info("branch: NewRequest failed", "error", err.Error())
		return
	}
	reqNr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	reqNr.Header.Set("Accept", "application/json")

	l.Logger.V(3).Info("branch: HTTP request created successfully")

	// send request.
	resp, err = l.execRequest(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: execRequest failed", "error", err.Error())
		return
	}

	l.Logger.V(3).Info("branch: execRequest succeeded")
	return
}

func (l APIImplement) execRequestOAuth(method string, baseUrl string, apiname string, apiKey string, queryParameter string) (resp interface{}, err error) {
	l.Logger.V(3).Info("start execRequestOAuth", "method", method, "url", baseUrl, "apiname", apiname)
	defer func() {
		l.Logger.V(3).Info("end execRequestOAuth", "err", err)
	}()

	// generate URL
	reqUrl := fmt.Sprintf(`%s/%s`, baseUrl, apiname)
	if queryParameter != "" {
		reqUrl = fmt.Sprintf(`%s?%s`, reqUrl, queryParameter)
	}
	l.Logger.V(3).Info("generated request URL", "url", reqUrl)

	// create request.
	var reqNr *http.Request
	reqNr, err = http.NewRequest(method, reqUrl, nil)
	if err != nil {
		l.Logger.V(2).Info("branch: NewRequest failed", "error", err.Error())
		return
	}
	// reqNr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	// reqNr.Header.Set("Accept", "application/json")
	parts := strings.Split(apiKey, ":")
	if len(parts) != 3 {
		klog.V(2).InfoS("branch: invalid API key format", "parts_count", len(parts))
		return
	}

	oauthConsumerKey := parts[0]
	oauthToken := parts[1]
	oauthSignature := "&" + parts[2]

	// other parameters
	oauthVersion := "1.0"
	oauthNonce := uuid.New().String()
	oauthTimestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// generate oauth header
	oauthHeader := fmt.Sprintf(
		`OAuth oauth_version="%s", oauth_signature_method="PLAINTEXT", oauth_consumer_key="%s", oauth_token="%s", oauth_signature="%s", oauth_nonce="%s", oauth_timestamp="%s"`,
		oauthVersion,
		url.QueryEscape(oauthConsumerKey),
		url.QueryEscape(oauthToken),
		url.QueryEscape(oauthSignature),
		oauthNonce,
		oauthTimestamp,
	)
	klog.V(3).InfoS("branch: OAuth header generated", "oauthNonce", oauthNonce, "oauthTimestamp", oauthTimestamp)

	reqNr.Header.Set("Authorization", oauthHeader)

	l.Logger.V(3).Info("branch: HTTP request created successfully")

	// send request.
	resp, err = l.execRequest(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: execRequest failed", "error", err.Error())
		return
	}

	l.Logger.V(3).Info("branch: execRequest succeeded")
	return
}

func (l APIImplement) execRequest(reqNr *http.Request) (resp interface{}, err error) {
	l.Logger.V(3).Info("start execRequest")
	defer func() {
		l.Logger.V(3).Info("end execRequest", "err", err)
	}()

	// logging for debug
	dump, err := httputil.DumpRequestOut(reqNr, true)
	if err == nil {
		l.Logger.V(3).Info("HTTP request dump", "dump", string(dump))
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
	l.Logger.V(3).Info("branch: HTTP client configured with TLS settings")

	// send request.
	respCd, err := client.Do(reqNr)
	if err != nil {
		l.Logger.V(2).Info("branch: HTTP request failed", "error", err.Error())
		return
	}
	defer func() { _ = respCd.Body.Close() }()

	l.Logger.V(3).Info("branch: HTTP request completed", "statusCode", respCd.StatusCode)

	// read response body.
	result, err := io.ReadAll(respCd.Body)
	if err != nil {
		l.Logger.V(2).Info("branch: response body read failed", "error", err.Error())
		return
	}

	l.Logger.V(3).Info("branch: response body read successfully", "bodySize", len(result))
	l.Logger.V(3).Info("branch: response body detail", "body", string(result))

	// HTTP status check
	if respCd.StatusCode < 200 || respCd.StatusCode >= 300 { // not 2xx -> error status
		l.Logger.V(2).Info("branch: HTTP status error", "statusCode", respCd.StatusCode, "responseSize", len(result))
		l.Logger.V(3).Info("branch: HTTP status error detail", "response", string(result))
		err = &CustomError{StatusCode: respCd.StatusCode, Message: string(result)}
		return
	}

	l.Logger.V(3).Info("branch: HTTP status successful", "statusCode", respCd.StatusCode)

	// extract JSON response
	resp, err = l.extracJsonResponse(result)
	if err != nil {
		l.Logger.V(2).Info("branch: JSON extraction failed", "error", err.Error())
		return
	}

	l.Logger.V(3).Info("branch: JSON extraction successful")
	return
}

func (l APIImplement) extracJsonResponse(ressponseBody []byte) (jsonResponse interface{}, err error) {
	l.Logger.V(3).Info("start extracJsonResponse", "bodySize", len(ressponseBody))
	defer func() {
		l.Logger.V(3).Info("end extracJsonResponse", "err", err)
	}()

	var result interface{}
	err = json.Unmarshal(ressponseBody, &result)
	if err != nil {
		l.Logger.V(2).Info("branch: JSON unmarshal failed", "error", err.Error())
		err = errors.New("response body is invalid for json format")
		return
	}

	l.Logger.V(3).Info("branch: JSON unmarshal successful")
	jsonResponse = result
	return
}

