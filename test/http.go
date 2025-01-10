// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// The MockRoundTripper implements the transport.RoundTripper interface.
// The transport.RoundTripper is a transport level mechanism part of every golang http.Client.
// By injecting the MockRoundTripper, you can control the responses of one or more calls by
// your component under test.
//
// Thus your component is making real HTTP calls, while the transport level returns your predefined responses.
// This enables no external dependencies in your tests, and allows you to test your code in isolation controlling all
// the failure or success modes.
//
// Example:
// func TestMyComponent(t *testing.T) {
//   mock := NewHttpMock()
//   mock.Expect("GET", "http://example.com", "Hello World", 200, nil)
//   mock.Expect("GET", "http://example.com", "", 500, errors.New("Internal Server Error"))
//
//   client := mock.HttpClient()
//   // inject the client to your component
//   myComponent := NewMyComponent(client)
//   // run your tests
//   results, err := myComponent.DoSomething()
//   // assert results
// }

type MockRoundTripper struct {
	Responses map[string][]*http.Response
	Errors    map[string][]error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	method := req.Method
	if responses, ok := m.Responses[method]; ok && len(responses) > 0 {
		response := responses[0]
		m.Responses[method] = responses[1:]
		if len(m.Responses[method]) == 0 {
			delete(m.Responses, method)
		}

		var err error
		if errors, ok := m.Errors[method]; ok && len(errors) > 0 {
			err = errors[0]
			m.Errors[method] = errors[1:]
			if len(m.Errors[method]) == 0 {
				delete(m.Errors, method)
			}
		}
		return response, err
	}

	errMsg := fmt.Sprintf("Not Mocked - Unexpected Call: %s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader(errMsg)),
		Header:     make(http.Header),
	}, nil
}

func (m *MockRoundTripper) Expect(method string, body string, status int, err error) {
	response := &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	m.Responses[method] = append(m.Responses[method], response)
	m.Errors[method] = append(m.Errors[method], err)
}

func NewHTTPMock() *MockRoundTripper {
	return &MockRoundTripper{
		Responses: make(map[string][]*http.Response),
		Errors:    make(map[string][]error),
	}
}

func (m *MockRoundTripper) HTTPClient() *http.Client {
	return &http.Client{
		Transport: m,
	}
}
