// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build unit
// +build unit

package validation_test

import (
	"strings"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/validation"
)

func TestValidateCloudAccountID(t *testing.T) {
	tests := []struct {
		input          string
		expectedOutput string
		expectError    bool
	}{
		{`"valid-123-id"`, "valid-123-id", false},
		{"invalid_id!", "", true},
		{"another-valid-id", "another-valid-id", false},
		{`"invalid!id"`, "", true},
		{"", "", true},
	}

	for _, test := range tests {
		output, err := validation.ValidateCloudAccountID(test.input)
		if test.expectError && err == nil {
			t.Errorf("Expected error for input '%s', but got none.", test.input)
		}
		if !test.expectError {
			if err != nil {
				t.Errorf("Did not expect error for input '%s', but got: %s", test.input, err.Error())
			}
			if output != test.expectedOutput {
				t.Errorf("For input '%s', expected output '%s', but got '%s'", test.input, test.expectedOutput, output)
			}
		}
	}
}

func TestValidateClusterName(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{"valid-cluster-name", false},
		{"invalid_cluster_name!", true},
		{"-invalidstart", true},
		{"invalidend-", true},
		{"a", false},
		{strings.Repeat("a", 253), false},
		{strings.Repeat("a", 254), true},
	}

	for _, test := range tests {
		err := validation.ValidateClusterName(test.input)
		if test.expectError && err == nil {
			t.Errorf("Expected error for cluster name '%s', but got none.", test.input)
		}
		if !test.expectError && err != nil {
			t.Errorf("Did not expect error for cluster name '%s', but got: %s", test.input, err.Error())
		}
	}
}
