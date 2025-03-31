// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package test_utils

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type WaitForLogInput struct {
	Container *testcontainers.Container
	Log       string
	Timeout   time.Duration
	Poll      time.Duration
}

// Polls the logs of the container to see if a `log` message exists. If the timeout is
// exceeded, an error returns.
func ContainerWaitForLog(ctx context.Context, input *WaitForLogInput) error {
	if input == nil {
		return fmt.Errorf("input is null")
	}
	if input.Container == nil {
		return fmt.Errorf("container is nil")
	}
	if input.Log == "" {
		return fmt.Errorf("log is empty")
	}

	return WaitForCondition(ctx, input.Timeout, input.Poll, func() (bool, error) {
		// read the logs
		reader, err := (*input.Container).Logs(ctx)
		if err != nil {
			return false, err
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(string(data)), "error") {
			return false, fmt.Errorf("error message found")
		}

		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(input.Log)) {
			return true, nil
		}

		return false, nil
	})
}

// Gets the external host of for a specific container based on what port testcontainers
// forwarded to on the host machine.
// This is only needed for external communications between application -> container.
// This is NOT needed for container -> container interactions on the same network
func ContainerExternalHost(ctx context.Context, container testcontainers.Container, port string) (*url.URL, error) {
	// get the host
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the container host name: %w", err)
	}

	// compose the nat port
	natPort, err := nat.NewPort("tcp", port)
	if err != nil {
		return nil, fmt.Errorf("failed to create the nat port: %w", err)
	}
	mappedPort, err := container.MappedPort(ctx, natPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create the mappedPort: %w", err)
	}

	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", host, mappedPort.Port()),
	}

	return &url, nil
}
