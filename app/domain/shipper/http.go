// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time" // Keep for HTTPinitialRetryDelay etc. if used in client config

	config "github.com/cloudzero/cloudzero-agent/app/config/gator" // Assuming this is for your span logger
	"github.com/cloudzero/cloudzero-agent/app/inspector"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Keep your constants, they are good for configuring the retryablehttp.Client
const (
	HTTPMaxRetries   = 10
	HTTPRetryWaitMax = time.Second * 30
)

// HTTPClient interface remains the same
type HTTPClient interface {
	// Do processes the request
	Do(req *http.Request) (*http.Response, error)
}

// ZerologRetryableHTTPAdapter adapts zerolog.Logger to retryablehttp.Logger
type ZerologRetryableHTTPAdapter struct {
	logger *zerolog.Logger
	level  zerolog.Level
}

// NewZerologRetryableHTTPAdapter creates a new adapter.
// Default level is Debug.
func NewZerologRetryableHTTPAdapter(logger *zerolog.Logger, level zerolog.Level) *ZerologRetryableHTTPAdapter {
	if logger == nil {
		defaultLogger := log.Logger // Get global zerolog logger or a default
		logger = &defaultLogger
	}
	return &ZerologRetryableHTTPAdapter{logger: logger, level: level}
}

func (a *ZerologRetryableHTTPAdapter) Error(msg string, keysAndValues ...interface{}) {
	a.logger.Error().Fields(retryableHTTPKVsToMap(keysAndValues...)).Msg(msg)
}

func (a *ZerologRetryableHTTPAdapter) Info(msg string, keysAndValues ...interface{}) {
	a.logger.Info().Fields(retryableHTTPKVsToMap(keysAndValues...)).Msg(msg)
}

func (a *ZerologRetryableHTTPAdapter) Debug(msg string, keysAndValues ...interface{}) {
	a.logger.Debug().Fields(retryableHTTPKVsToMap(keysAndValues...)).Msg(msg)
}

func (a *ZerologRetryableHTTPAdapter) Warn(msg string, keysAndValues ...interface{}) {
	a.logger.Warn().Fields(retryableHTTPKVsToMap(keysAndValues...)).Msg(msg)
}

// retryableHTTPKVsToMap converts go-retryablehttp's key-value pairs to a map for zerolog.
func retryableHTTPKVsToMap(keysAndValues ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			if key, ok := keysAndValues[i].(string); ok {
				m[key] = keysAndValues[i+1]
			}
		}
	}
	return m
}

// Ensure it implements the LeveledLogger interface if you want more control
var _ retryablehttp.LeveledLogger = (*ZerologRetryableHTTPAdapter)(nil)

func NewHTTPClient(ctx context.Context, s *config.Settings) *retryablehttp.Client {
	httpClient := retryablehttp.NewClient()
	httpClient.Logger = NewZerologRetryableHTTPAdapter(log.Ctx(ctx), log.Ctx(ctx).GetLevel())
	httpClient.HTTPClient = &http.Client{
		Timeout: s.Cloudzero.SendTimeout,
	}
	httpClient.RetryMax = s.Cloudzero.HTTPMaxRetries
	httpClient.RetryWaitMax = s.Cloudzero.HTTPMaxWait

	httpClient.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		if resp == nil {
			return nil, errors.Join(fmt.Errorf("giving up after %d attempt(s): %w", numTries, err), ErrHTTPRequestFailed)
		}

		// drain the http body
		defer resp.Body.Close()
		if _, err2 := io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); err2 != nil {
			log.Ctx(ctx).Err(err2).Msg("error reading response body")
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.Join(err, ErrUnauthorized)
		}

		return nil, errors.Join(fmt.Errorf("giving up after %d attempt(s): %w", numTries, err), ErrHTTPRequestFailed)
	}

	return httpClient
}

func InspectHTTPResponse(ctx context.Context, resp *http.Response) error {
	var err error
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		err = ErrUnauthorized
	}

	switch {
	// request was successful
	case resp.StatusCode < http.StatusOK+100:
		break
	// unauthorized errors
	case resp.StatusCode == http.StatusForbidden:
	case resp.StatusCode == http.StatusUnauthorized:
		err = ErrUnauthorized
	// default case is unknown
	default:
		err = ErrHTTPUnknown
	}

	// inspect the response
	i := inspector.New()
	if err2 := i.Inspect(ctx, resp, *log.Ctx(ctx)); err2 != nil {
		err = errors.Join(err, err2)
	}

	// consume the body if there is an error
	if err != nil {
		defer resp.Body.Close()
		if _, err2 := io.Copy(io.Discard, io.LimitReader(resp.Body, 4096)); err2 != nil {
			log.Ctx(ctx).Err(err2).Msg("error reading response body")
		}
	}

	return err
}
