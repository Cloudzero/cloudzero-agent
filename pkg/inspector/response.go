// Package inspector provides a way to inspect HTTP responses from the CloudZero
// API to diagnose issues.
package inspector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// responseData is a wrapper around an HTTP response that provides a convenient
// cache of a few basic operations.
type responseData struct {
	resp     *http.Response
	lock     sync.Mutex
	bodyData []byte
	jsonData any
}

// IsJSON returns true if the response body is JSON (according to the
// Content-Type header).
func (resp *responseData) IsJSON() bool {
	if resp.jsonData != nil {
		return true
	}

	switch strings.ToLower(resp.resp.Header.Get("Content-Type")) {
	case "application/json":
		fallthrough
	case "text/json":
		return true
	}

	return false
}

// body returns the body of the response as a byte slice.
//
// This is basically a wrapper around io.ReadAll(resp.resp.body) that caches the
// result so it can be accessed multiple times, while also replacing the
// response body so the data can be read again when passing the response back to
// the HTTP client.
func (resp *responseData) body() []byte {
	if resp.bodyData != nil {
		return resp.bodyData
	}

	if resp.resp.Body != nil {
		func() {
			resp.lock.Lock()
			defer resp.lock.Unlock()

			body, err := io.ReadAll(resp.resp.Body)
			if err != nil {
				return
			}

			resp.resp.Body = io.NopCloser(bytes.NewBuffer(body))
			resp.bodyData = body
		}()
	}

	return resp.bodyData
}

// jsonBody returns the decoded JSON body of the response.
func (resp *responseData) jsonBody() (any, error) {
	if resp.jsonData != nil {
		return resp.jsonData, nil
	}

	var jsonData any
	err := json.Unmarshal(resp.body(), &jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body to JSON: %w", err)
	}

	func() {
		resp.lock.Lock()
		defer resp.lock.Unlock()

		resp.jsonData = jsonData
	}()

	return resp.jsonData, nil
}

func (resp *responseData) JSONMatch(query string) (bool, error) {
	if !resp.IsJSON() {
		return false, nil
	}

	jsonData, err := resp.jsonBody()
	if err != nil {
		return false, err
	}

	return gojqCache.JSONMatch(query, jsonData)
}
