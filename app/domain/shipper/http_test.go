// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock HTTP Client
type MockHTTPClient struct {
	responses []*http.Response
	errors    []error
	callCount int32
	delays    []time.Duration // Optional delays for simulating slow responses
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{}
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	callIndex := int(atomic.AddInt32(&m.callCount, 1) - 1)

	// Simulate delay if specified
	if callIndex < len(m.delays) && m.delays[callIndex] > 0 {
		time.Sleep(m.delays[callIndex])
	}

	// Return error if specified
	if callIndex < len(m.errors) && m.errors[callIndex] != nil {
		var resp *http.Response
		if callIndex < len(m.responses) {
			resp = m.responses[callIndex]
		}
		return resp, m.errors[callIndex]
	}

	// Return response if available
	if callIndex < len(m.responses) && m.responses[callIndex] != nil {
		return m.responses[callIndex], nil
	}

	// Default successful response
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("success")),
		Header:     make(http.Header),
	}, nil
}

func (m *MockHTTPClient) Reset() {
	atomic.StoreInt32(&m.callCount, 0)
	m.responses = nil
	m.errors = nil
	m.delays = nil
}

func (m *MockHTTPClient) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func (m *MockHTTPClient) AddResponse(statusCode int, body string) {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	m.responses = append(m.responses, resp)
}

func (m *MockHTTPClient) AddResponseWithHeaders(statusCode int, body string, headers map[string]string) {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	m.responses = append(m.responses, resp)
}

func (m *MockHTTPClient) AddError(err error) {
	m.errors = append(m.errors, err)
	m.responses = append(m.responses, nil)
}

func (m *MockHTTPClient) AddDelay(delay time.Duration) {
	m.delays = append(m.delays, delay)
}

// Helper functions
func createTestRequest() (*http.Request, error) {
	return http.NewRequest("GET", "http://example.com/test", nil)
}

func createTestLogger() *zerolog.Logger {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()
	return &logger
}

func createFailingRequestFunc(failCount int) func() (*http.Request, error) {
	callCount := 0
	return func() (*http.Request, error) {
		callCount++
		if callCount <= failCount {
			return nil, fmt.Errorf("request preparation failed on call %d", callCount)
		}
		return createTestRequest()
	}
}

// Test cases
func TestShipper_Unit_SendHTTPRequest_ImmediateSuccess(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, client.GetCallCount())

	// Verify response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "success", string(body))
	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_SuccessAfterRetries(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(500, "server error")
	client.AddResponse(502, "bad gateway")
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	start := time.Now()
	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, client.GetCallCount())

	// Should have taken at least some time due to retry delays
	assert.True(t, duration > 500*time.Millisecond, "Expected some delay for retries")

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_AllRetriesExhausted_ServerError(t *testing.T) {
	client := NewMockHTTPClient()

	// Add maxRetries number of 500 responses
	for i := 0; i < shipper.HTTPmaxRetries; i++ {
		client.AddResponse(500, "server error")
	}

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed with status 500")
	assert.Contains(t, err.Error(), fmt.Sprintf("after %d attempts", shipper.HTTPmaxRetries))
	assert.Equal(t, shipper.HTTPmaxRetries, client.GetCallCount())

	require.NotNil(t, resp)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestShipper_Unit_SendHTTPRequest_NonRetryableClientError(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"Bad Request", 400},
		{"Unauthorized", 401},
		{"Forbidden", 403},
		{"Not Found", 404},
		{"Method Not Allowed", 405},
		{"Unprocessable Entity", 422},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewMockHTTPClient()
			client.AddResponse(tc.statusCode, "client error")

			logger := createTestLogger()
			ctx := context.Background()

			resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("request failed with status %d", tc.statusCode))
			assert.Equal(t, 1, client.GetCallCount()) // Should not retry client errors

			require.NotNil(t, resp)
			assert.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}

func TestShipper_Unit_SendHTTPRequest_RetryAfterHeader(t *testing.T) {
	client := NewMockHTTPClient()

	// First response with Retry-After header
	client.AddResponseWithHeaders(429, "too many requests", map[string]string{
		"Retry-After": "2",
	})
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	start := time.Now()
	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, client.GetCallCount())

	// Should have waited for the Retry-After duration (2 seconds + jitter)
	assert.True(t, duration >= 2*time.Second, "Expected to wait for Retry-After duration")

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_RetryAfterHeader_InvalidValue(t *testing.T) {
	client := NewMockHTTPClient()

	// Response with invalid Retry-After header
	client.AddResponseWithHeaders(429, "too many requests", map[string]string{
		"Retry-After": "invalid",
	})
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, client.GetCallCount())

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_NetworkErrors(t *testing.T) {
	client := NewMockHTTPClient()

	networkErr := errors.New("network unreachable")
	client.AddError(networkErr)
	client.AddError(networkErr)
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 3, client.GetCallCount())

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_NetworkErrorsExhausted(t *testing.T) {
	client := NewMockHTTPClient()

	networkErr := errors.New("network unreachable")
	for i := 0; i < shipper.HTTPmaxRetries; i++ {
		client.AddError(networkErr)
	}

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed after")
	assert.Contains(t, err.Error(), "attempts with client error")
	assert.Equal(t, shipper.HTTPmaxRetries, client.GetCallCount())
	assert.Nil(t, resp)
}

func TestShipper_Unit_SendHTTPRequest_ContextCancellation_BeforeRequest(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 0, client.GetCallCount())
	assert.Nil(t, resp)
}

func TestShipper_Unit_SendHTTPRequest_ContextCancellation_DuringRetry(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(500, "server error")

	logger := createTestLogger()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay to simulate cancellation during retry sleep
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, client.GetCallCount())

	if resp != nil {
		assert.Equal(t, 500, resp.StatusCode)
	}
}

func TestShipper_Unit_SendHTTPRequest_ContextCancellation_DuringRequest(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddError(context.Canceled)

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 1, client.GetCallCount()) // Should not retry on context cancellation
	assert.Nil(t, resp)
}

func TestShipper_Unit_SendHTTPRequest_RequestTimeout(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddError(context.DeadlineExceeded)

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Equal(t, 1, client.GetCallCount()) // Should not retry on deadline exceeded
	assert.Nil(t, resp)
}

func TestShipper_Unit_SendHTTPRequest_RequestPreparationFailure(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	// Function that fails first 2 times, then succeeds
	failingRequestFunc := createFailingRequestFunc(2)

	resp, err := shipper.SendHTTPRequest(ctx, client, failingRequestFunc, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, client.GetCallCount()) // Only one successful HTTP call

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_RequestPreparationAlwaysFails(t *testing.T) {
	client := NewMockHTTPClient()

	logger := createTestLogger()
	ctx := context.Background()

	failingRequestFunc := func() (*http.Request, error) {
		return nil, errors.New("request preparation always fails")
	}

	resp, err := shipper.SendHTTPRequest(ctx, client, failingRequestFunc, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to prepare request for attempt")
	assert.Equal(t, 0, client.GetCallCount()) // No HTTP calls made
	assert.Nil(t, resp)
}

func TestShipper_Unit_SendHTTPRequest_ResponseWithError(t *testing.T) {
	client := NewMockHTTPClient()

	// Simulate case where HTTP client returns both response and error
	client.responses = []*http.Response{
		{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader("error response")),
			Header:     make(http.Header),
		},
	}
	client.errors = []error{errors.New("some error but response available")}

	// Add a successful second response
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 2, client.GetCallCount())

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_NilLogger(t *testing.T) {
	client := NewMockHTTPClient()
	client.AddResponse(200, "success")

	ctx := context.Background()

	// Test with nil logger - should not panic
	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, client.GetCallCount())

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_ExponentialBackoff(t *testing.T) {
	client := NewMockHTTPClient()

	// Add multiple server errors to test backoff
	for i := 0; i < 4; i++ {
		client.AddResponse(500, "server error")
	}
	client.AddResponse(200, "success")

	logger := createTestLogger()
	ctx := context.Background()

	start := time.Now()
	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 5, client.GetCallCount())

	// With exponential backoff: 1s + 2s + 4s + 8s = 15s minimum (plus jitter)
	expectedMinDuration := shipper.HTTPinitialRetryDelay + 2*shipper.HTTPinitialRetryDelay + 4*shipper.HTTPinitialRetryDelay + 8*shipper.HTTPinitialRetryDelay
	assert.True(t, duration >= expectedMinDuration*8/10, // Allow some tolerance for jitter
		"Expected duration >= %v, got %v", expectedMinDuration*8/10, duration)

	resp.Body.Close()
}

func TestShipper_Unit_SendHTTPRequest_MaxDelayClamp(t *testing.T) {
	client := NewMockHTTPClient()

	// Add enough retries to exceed maxRetryDelay
	for i := 0; i < shipper.HTTPmaxRetries; i++ {
		client.AddResponse(500, "server error")
	}

	logger := createTestLogger()
	ctx := context.Background()

	start := time.Now()
	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)
	duration := time.Since(start)

	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, shipper.HTTPmaxRetries, client.GetCallCount())

	// Should not exceed reasonable bounds due to maxRetryDelay clamping
	maxExpectedDuration := time.Duration(shipper.HTTPmaxRetries) * shipper.HTTPmaxRetryDelay * 2 // Allow generous buffer
	assert.True(t, duration < maxExpectedDuration,
		"Duration %v exceeded maximum expected %v", duration, maxExpectedDuration)
}

func TestShipper_Unit_SendHTTPRequest_SpecificServerErrorCodes(t *testing.T) {
	retryableStatusCodes := []int{500, 501, 502, 503, 504, 505, 507, 508, 510, 511}

	for _, statusCode := range retryableStatusCodes {
		t.Run(fmt.Sprintf("StatusCode_%d", statusCode), func(t *testing.T) {
			client := NewMockHTTPClient()
			client.AddResponse(statusCode, "server error")
			client.AddResponse(200, "success")

			logger := createTestLogger()
			ctx := context.Background()

			resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, 2, client.GetCallCount()) // Should have retried

			resp.Body.Close()
		})
	}
}

type trackingBody struct {
	*strings.Reader
	closed bool
}

func (tb *trackingBody) Close() error {
	tb.closed = true
	return nil
}

// Test for memory leaks - ensures response bodies are properly closed
func TestShipper_Unit_SendHTTPRequest_BodyCleanup(t *testing.T) {
	client := NewMockHTTPClient()

	// Create responses with tracking bodies
	body1 := &trackingBody{Reader: strings.NewReader("error1")}
	body2 := &trackingBody{Reader: strings.NewReader("error2")}
	body3 := &trackingBody{Reader: strings.NewReader("success")}

	client.responses = []*http.Response{
		{StatusCode: 500, Body: body1, Header: make(http.Header)},
		{StatusCode: 502, Body: body2, Header: make(http.Header)},
		{StatusCode: 200, Body: body3, Header: make(http.Header)},
	}

	logger := createTestLogger()
	ctx := context.Background()

	resp, err := shipper.SendHTTPRequest(ctx, client, createTestRequest, logger)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify that failed response bodies were closed
	assert.True(t, body1.closed, "First response body should be closed")
	assert.True(t, body2.closed, "Second response body should be closed")

	// Success response body should not be closed (caller's responsibility)
	assert.False(t, body3.closed, "Success response body should not be closed")

	// Clean up
	resp.Body.Close()
	assert.True(t, body3.closed, "Success response body should be closed after caller closes it")
}
