// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	HTTPmaxRetries        = 5
	HTTPinitialRetryDelay = 1 * time.Second
	HTTPmaxRetryDelay     = 30 * time.Second
	HTTPjitterFactor      = 0.1 // 10% jitter
)

// HTTPClient interface for dependency injection and testing
type HTTPClient interface {
	// Do sends the http request
	Do(req *http.Request) (*http.Response, error)
}

// SendHTTPRequest performs an HTTP requests with retry logic
func SendHTTPRequest(
	ctx context.Context,
	client HTTPClient,
	getRequest func() (*http.Request, error),
	logger *zerolog.Logger,
) (*http.Response, error) {
	var finalResp *http.Response
	var lastErr error

	if logger == nil {
		logger = log.Ctx(ctx)
	}

	currentDelay := HTTPinitialRetryDelay
	for attempt := range HTTPmaxRetries {
		// check if the current context has an error in it
		if ctx.Err() != nil {
			logger.Warn().Err(ctx.Err()).Msg("Context cancelled before attempt")
			lastErr = ctx.Err()
			return finalResp, lastErr
		}

		if attempt > 0 {
			logger.Debug().Int("attempt", attempt+1).Msg("Re-trying HTTP request")

			// calculate some random jitter to occur
			jitter := time.Duration(rand.Float64() * float64(currentDelay) * HTTPjitterFactor) //nolint:gosec // rand impl not important
			sleepDuration := currentDelay + jitter

			logger.Debug().Int("attempt", attempt+1).Dur("sleepDuration", sleepDuration).Msg("Sleeping ...")

			// ensure the context does not get cancelled while sleeping
			select {
			case <-time.After(sleepDuration):
				// continue
			case <-ctx.Done():
				logger.Warn().Err(ctx.Err()).Msg("Context cancelled during retry sleep")
				lastErr = ctx.Err()
				return finalResp, lastErr
			}

			// exponential backoff with clamp
			currentDelay *= 2
			if currentDelay > HTTPmaxRetryDelay {
				currentDelay = HTTPmaxRetryDelay
			}
		}

		// get a fresh request every retry
		req, err := getRequest()
		if err != nil {
			logger.Err(err).Msg("Failed to prepare request for attempt")
			lastErr = fmt.Errorf("failed to prepare request for attempt %d: %w", attempt+1, err)
			if attempt < HTTPmaxRetries-1 {
				continue
			}

			// all re-tries used
			return finalResp, lastErr
		}

		// send the http request attempt
		logger.Debug().Int("attempt", attempt+1).Msg("Sending HTTP request ...")
		resp, err := client.Do(req)
		if resp != nil {
			finalResp = resp
		}

		// store the last error seen from http.Do
		lastErr = err

		if err != nil {
			logger.Err(err).Int("attempt", attempt+1).Msg("HTTPClient.Do error")
			if resp != nil {
				// consume the body
				body, ierr := io.ReadAll(resp.Body)
				if ierr != nil {
					logger.Warn().Msg("failed to read the response body")
				} else {
					logger.Warn().Str("responseBody", string(body)).Msg("Response Body")
				}
				resp.Body.Close()
			}

			// if the context was cancelled, do not retry
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return finalResp, err
			}

			// re-try
			if attempt < HTTPmaxRetries-1 {
				continue
			}

			// propagate error up
			return finalResp, errors.Join(ErrHTTPRequestFailed, fmt.Errorf("request failed after %d attempts with client error: %w", attempt+1, err))
		}

		// check for a successful request
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusOK+100 {
			logger.Debug().Int("attempt", attempt+1).Int("status", resp.StatusCode).Msg("HTTP request successful")
			finalResp = resp
			return finalResp, nil
		}

		logger.Warn().Int("attempt", attempt+1).Int("status", resp.StatusCode).Msg("HTTP request returned non-2xx status")

		// check for a forbidden error
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			// do not retry with invalid auth credentials
			return finalResp, errors.Join(ErrUnauthorized, fmt.Errorf("request failed with status %d", resp.StatusCode))
		}

		// if we receive a 429, this could be the downstream attempting to slow us down
		if resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					retryDelay := time.Duration(seconds) * time.Second
					if retryDelay <= HTTPmaxRetryDelay {
						// override current delay with server-provided delay
						currentDelay = retryDelay
					}
				}
			}

			// retry regardless with this status code
			continue
		}

		// if a server error, then we can retry the request
		isRetryableStatus := resp.StatusCode >= http.StatusInternalServerError

		if isRetryableStatus && attempt < HTTPmaxRetries-1 {
			// consume the body and retry
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Warn().Msg("failed to read the response body")
			} else {
				logger.Warn().Str("responseBody", string(body)).Msg("Response body")
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("http status %d on attempt %d", resp.StatusCode, attempt+1)
			continue
		}

		// if here, then the error was not retryable or we ran out of attempts
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			logger.Warn().Msg("failed to discard the response body")
		}
		resp.Body.Close()
		return finalResp, errors.Join(ErrHTTPUnknown, fmt.Errorf("request failed with status %d after %d attempts", resp.StatusCode, attempt+1))
	}

	if lastErr == nil {
		// check if there was a last unhandled error
		if finalResp != nil && (finalResp.StatusCode < http.StatusOK || finalResp.StatusCode >= http.StatusOK+100) {
			lastErr = errors.Join(ErrHTTPUnknown, fmt.Errorf("request completed with unhandled status: %d", finalResp.StatusCode))
		} else if finalResp == nil {
			lastErr = errors.Join(ErrHTTPUnknown, errors.New("retries exhausted without a final response or error"))
		}
	}
	return finalResp, lastErr
}

// SendHTTPRequest sends a provided HTTP request obtained through the `getRequest` function.
//
// This is a wrapper around the default method `SendHTTPRequest`.
//
// Compose the HTTP request inside of getRequest to ensure that if an HTTP request
// is retried, a new request object is made with a fresh body.
func (m *MetricShipper) SendHTTPRequest(
	ctx context.Context,
	name string,
	getRequest func() (*http.Request, error),
) (*http.Response, error) {
	var finalResp *http.Response

	// wrap in a span
	spanErr := m.metrics.SpanCtx(ctx, name, func(currentCtx context.Context, id string) error {
		logger := instr.SpanLogger(currentCtx, id, func(zc zerolog.Context) zerolog.Context {
			return zc.Str("httpRequestName", name)
		})

		resp, err := SendHTTPRequest(
			currentCtx,
			m.HTTPClient,
			getRequest,
			&logger,
		)

		finalResp = resp
		return err
	})

	if spanErr != nil {
		return finalResp, spanErr
	}

	// should not happen, but you never know
	if finalResp == nil {
		return nil, errors.Join(ErrHTTPRequestFailed, errors.New("request succeeded but no response available"))
	}

	// pass the response to the caller
	return finalResp, nil
}
