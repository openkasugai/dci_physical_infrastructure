package canonical_maas

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maas_module/internal/server/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

// HTTPClient interface for dependency injection
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// struct of Canonical MaaS API
type CanonicalMaasAPIImple struct {
	Logger klog.Logger
	Client HTTPClient
	AccessUrl string
	ApiKey    string
}

// NewCanonicalMaasAPIImple creates a new instance with default HTTP client
func NewCanonicalMaasAPIImple(logger klog.Logger, accessUrl string, apiKey string) *CanonicalMaasAPIImple {
	return &CanonicalMaasAPIImple{
		Logger: logger,
		Client: &http.Client{},
		AccessUrl: accessUrl,
		ApiKey: apiKey,
	}
}

// API execution
func (l CanonicalMaasAPIImple) APIExecute(ctx context.Context, method string, apiname string, reqBody string, withRetry bool) (statusCode int, jsonData []byte, err error) {
	klog.V(2).InfoS("API execution", "method", method, "apiname", apiname, "withRetry", withRetry)
	klog.V(3).InfoS("start APIExecute", "method", method, "apiname", apiname, "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("API completed", "statusCode", statusCode, "err", err)
		klog.V(3).InfoS("end APIExecute", "statusCode", statusCode, "jsonData", string(jsonData), "err", err)
	}()

	// create URL.
	url := l.AccessUrl + apiname
	klog.V(3).InfoS("branch: generated request URL", "url", url)

	// Determine retry parameters
	maxAttempts := 1
	retryInterval := time.Duration(0)
	if withRetry {
		cfg := utils.GetConfig()
		if cfg != nil && cfg.MaasApiRetryCount > 0 {
			maxAttempts = cfg.MaasApiRetryCount + 1
		}
		if cfg != nil && cfg.MaasApiRetryIntervalSeconds > 0 {
			retryInterval = time.Duration(cfg.MaasApiRetryIntervalSeconds) * time.Second
		}
	}

	// execute http request with retry.
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			klog.V(2).InfoS("retrying API request", "attempt", attempt, "maxAttempts", maxAttempts, "method", method, "apiname", apiname)
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			case <-time.After(retryInterval):
			}
		}

		var execErr error
		statusCode, jsonData, execErr = l.execRequest(method, url, reqBody)
		if execErr != nil {
			klog.V(2).InfoS("branch: request execution failed", "attempt", attempt, "error", execErr.Error())
			err = execErr
			continue
		}

		// check HTTP status code: retry on server errors (5xx) or other non-2xx codes
		if statusCode >= 200 && statusCode < 300 {
			klog.V(2).InfoS("branch: request execution successful", "statusCode", statusCode)
			err = nil
			return
		}

		klog.V(2).InfoS("branch: request returned non-2xx status", "attempt", attempt, "statusCode", statusCode)
		err = &utils.HttpError{StatusCode: statusCode, Message: string(jsonData)}
	}

	return
}

func (l CanonicalMaasAPIImple) execRequest(method string, reqUrl string, reqBody string) (statusCode int, resp []byte, err error) {
	klog.V(2).InfoS("HTTP request", "method", method, "url", reqUrl)
	klog.V(3).InfoS("start execRequest", "method", method, "reqUrl", reqUrl, "reqBody", reqBody)
	defer func() {
		klog.V(2).InfoS("HTTP response", "statusCode", statusCode, "err", err)
		klog.V(3).InfoS("end execRequest", "statusCode", statusCode, "resp", string(resp), "err", err)
	}()

	parts := strings.Split(l.ApiKey, ":")
	if len(parts) != 3 {
		klog.V(2).InfoS("branch: invalid API key format", "parts_count", len(parts))
		err = &utils.EnvError{Message: "invalid API key format"}
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

	// create request.
	var reqNr *http.Request
	if reqBody == "" {
		klog.V(3).InfoS("branch: creating request without body")
		reqNr, err = http.NewRequest(method, reqUrl, nil)
	} else {
		klog.V(3).InfoS("branch: creating request with body")
		reqNr, err = http.NewRequest(method, reqUrl, bytes.NewBuffer([]byte(reqBody)))
	}
	if err != nil {
		klog.V(2).InfoS("branch: request creation failed", "error", err.Error())
		err = &utils.EnvError{Message: err.Error()}
		klog.Error(err, "Failed to create HTTP request")
		return
	}
	reqNr.Header.Set("Authorization", oauthHeader)
	if reqBody != "" {
		reqNr.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// logging for debug
	dump, err := httputil.DumpRequestOut(reqNr, true)
	if err == nil {
		klog.V(3).InfoS("HTTP request details", "dump", string(dump))
	} else {
		klog.V(3).InfoS("branch: failed to dump request", "error", err.Error())
	}

	// send request.
	respCd, err := l.Client.Do(reqNr)
	if err != nil {
		klog.V(2).InfoS("branch: HTTP request failed", "error", err.Error())
		err = &utils.EnvError{Message: err.Error()}
		klog.Error(err, "HTTP request execution failed")
		return
	}
	defer func() { _ = respCd.Body.Close() }()

	klog.V(2).InfoS("branch: HTTP request successful", "statusCode", respCd.StatusCode)

	// read response body.
	result, err := io.ReadAll(respCd.Body)
	if err != nil {
		klog.V(2).InfoS("branch: response body read failed", "error", err.Error())
		err = &utils.EnvError{Message: err.Error()}
		klog.Error(err, "Failed to read response body")
		return
	}

	klog.V(2).InfoS("branch: response body read successful", "length", len(result))
	statusCode = respCd.StatusCode
	resp = result
	return
}
