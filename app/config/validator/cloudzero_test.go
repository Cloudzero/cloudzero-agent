// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/stretchr/testify/assert"
)

func TestCloudzero_Validate(t *testing.T) {
	const testValue = "the-cz-api-key"

	// make a temp file with the API key
	tmpFile, err := os.CreateTemp(t.TempDir(), "cloudzero")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.Write([]byte(testValue))
	assert.NoError(t, err)
	assert.NoError(t, tmpFile.Close()) // Close the file before we try to read it

	tests := []struct {
		name    string
		config  config.Cloudzero
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: config.Cloudzero{
				Host:            "http://api.cloudzero.com",
				CredentialsFile: tmpFile.Name(),
			},
			wantErr: false,
		},
		{
			name: "Empty host",
			config: config.Cloudzero{
				Host:            "",
				CredentialsFile: tmpFile.Name(),
			},
			wantErr: true,
		},
		{
			name: "Missing API key file",
			config: config.Cloudzero{
				Host:            "http://api.cloudzero.com",
				CredentialsFile: "/path/to/nonexistent/file",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// run the validation, and if we want an error, make sure we get one
			// if not - then validate we got the right key
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, testValue, tt.config.Credential)
		})
	}
}
