package config_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
)

func TestLogging_Validate(t *testing.T) {
	tests := []struct {
		name    string
		logging config.Logging
		wantErr bool
	}{
		{
			name: "Valid logging configuration",
			logging: config.Logging{
				Level: "debug", Location: "./cloudzero-agent-validator.log",
			},
			wantErr: false,
		},
		{
			name: "Empty logging level",
			logging: config.Logging{
				Location: "cloudzero-agent-validator.log",
			},
			wantErr: false,
		},
		{
			name: "Invalid logging level default to info",
			logging: config.Logging{
				Level: "bogus", Location: "cloudzero-agent-validator.log",
			},
			wantErr: false,
		},
		{
			name: "Invalid log directory",
			logging: config.Logging{
				Location: "/invalid/directory/cloudzero-agent-validator.log",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.logging.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
