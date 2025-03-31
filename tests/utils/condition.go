// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"time"
)

// WaitForCondition waits for a specific condition to return true.
func WaitForCondition(
	ctx context.Context,
	timeout time.Duration,
	poll time.Duration,
	condition func() (bool, error),
) error {
	// fix zero values
	if timeout == 0 {
		timeout = time.Duration(1) * time.Minute
	}
	if poll == 0 {
		poll = time.Duration(1) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout reached, condition not met")
		case <-ticker.C:
			passed, err := condition()
			if err != nil {
				return err
			}
			if passed {
				return nil
			}
		}
	}
}
