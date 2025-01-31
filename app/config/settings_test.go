//go:build unit
// +build unit

package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
)

func TestCloudzeroSettings_Defaults(t *testing.T) {
	_, err := config.NewSettings("testdata/default_config.yaml")
	assert.NoError(t, err)
}

func TestCloudzeroSettings_APIKey(t *testing.T) {
	settings := config.Settings{
		Cloudzero: config.Cloudzero{
			APIKeyPath: "testdata/api_key.txt",
		},
	}
	err := settings.SetAPIKey()
	assert.NoError(t, err)
	assert.Equal(t, "test-api-key", settings.GetAPIKey())
}

func TestCloudzeroSettings_InvalidAPIKeyPath(t *testing.T) {
	settings := config.Settings{
		Cloudzero: config.Cloudzero{
			APIKeyPath: "testdata/invalid_api_key.txt",
		},
	}

	assert.Error(t, settings.SetAPIKey())

	settings = config.Settings{
		Cloudzero: config.Cloudzero{
			APIKeyPath: "testdata/invalid.file",
		},
	}
	assert.Error(t, settings.SetAPIKey())
}

func TestSettings_Validate(t *testing.T) {
	tests := []struct {
		name     string
		settings config.Settings
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "123456789012",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
				Server: config.Server{
					Mode: "http",
					Port: 8080,
				},
				Database: config.Database{
					StoragePath: "testdata",
					MaxRecords:  1000000,
					Compress:    true,
				},
				Cloudzero: config.Cloudzero{
					APIKeyPath:   "testdata/api_key.txt",
					SendInterval: 60 * time.Second,
					SendTimeout:  10 * time.Second,
					Host:         "api.cloudzero.com",
				},
			},
			wantErr: false,
		},
		{
			name: "empty org ID",
			settings: config.Settings{
				OrganizationID: "",
				CloudAccountID: "123456789012",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "empty cloud account ID",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "empty region",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "123456789012",
				Region:         "",
				ClusterName:    "test-cluster",
			},
			wantErr: true,
		},
		{
			name: "empty cluster name",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "123456789012",
				Region:         "us-east-1",
				ClusterName:    "",
			},
			wantErr: true,
		},
		{
			name: "invalid database storage path",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "123456789012",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
				Database: config.Database{
					StoragePath: "invalid_path",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid API key path",
			settings: config.Settings{
				OrganizationID: "testorg",
				CloudAccountID: "123456789012",
				Region:         "us-east-1",
				ClusterName:    "test-cluster",
				Cloudzero: config.Cloudzero{
					APIKeyPath: "invalid_path",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDatabase_Validate(t *testing.T) {
	tests := []struct {
		name     string
		database config.Database
		wantErr  bool
	}{
		{
			name: "valid database settings",
			database: config.Database{
				StoragePath: "testdata",
				MaxRecords:  1000000,
				Compress:    true,
			},
			wantErr: false,
		},
		{
			name: "invalid storage path",
			database: config.Database{
				StoragePath: "invalid_path",
			},
			wantErr: true,
		},
		{
			name: "negative max records",
			database: config.Database{
				StoragePath: "testdata",
				MaxRecords:  -1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.database.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		server  config.Server
		wantErr bool
	}{
		{
			name: "valid server settings",
			server: config.Server{
				Mode: "http",
				Port: 8080,
			},
			wantErr: false,
		},
		{
			name: "empty mode",
			server: config.Server{
				Mode: "",
				Port: 8080,
			},
			wantErr: false,
		},
		{
			name: "zero port",
			server: config.Server{
				Mode: "http",
				Port: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCloudzeroSettings_Validate(t *testing.T) {
	tests := []struct {
		name     string
		settings config.Cloudzero
		wantErr  bool
	}{
		{
			name: "valid cloudzero settings",
			settings: config.Cloudzero{
				APIKeyPath:   "testdata/api_key.txt",
				SendInterval: 60 * time.Second,
				SendTimeout:  10 * time.Second,
				Host:         "api.cloudzero.com",
			},
			wantErr: false,
		},
		{
			name: "empty API key path",
			settings: config.Cloudzero{
				APIKeyPath: "",
			},
			wantErr: true,
		},
		{
			name: "invalid API key path",
			settings: config.Cloudzero{
				APIKeyPath: "invalid_path",
			},
			wantErr: true,
		},
		{
			name: "empty host",
			settings: config.Cloudzero{
				APIKeyPath: "testdata/api_key.txt",
				Host:       "",
			},
			wantErr: false,
		},
		{
			name: "negative send interval",
			settings: config.Cloudzero{
				APIKeyPath:   "testdata/api_key.txt",
				SendInterval: -1,
			},
			wantErr: false,
		},
		{
			name: "negative send timeout",
			settings: config.Cloudzero{
				APIKeyPath:  "testdata/api_key.txt",
				SendTimeout: -1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
