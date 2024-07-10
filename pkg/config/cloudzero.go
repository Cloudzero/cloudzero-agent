package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pkg/errors"
)

// Cloudzero is the configuration for the CloudZero checker
type Cloudzero struct {
	Host             string `yaml:"host" default:"https://api.cloudzero.com" env:"CZ_HOST" env-description:"CloudZero API host"`
	Credential       string
	CredentialsFile  string `yaml:"credentials_file" default:"/etc/config/prometheus/secrets/value" env:"CZ_CREDENTIALS_FILE" env-description:"API Access Token file while running in a container"`
	DisableTelemetry bool   `yaml:"disable_telemetry" default:"false" env:"CZ_DISABLE_TELEMETRY" env-description:"Disable telemetry"`
}

func (s *Cloudzero) Validate() error {
	if s.Host == "" {
		return errors.New(ErrNoCloudZeroHostMsg)
	}
	if !isValidURL(s.Host) {
		return fmt.Errorf("URL format invalid: %s", s.Host)
	}

	location, err := absFilePath(s.CredentialsFile)
	if err != nil {
		return err
	}
	s.CredentialsFile = location

	if _, err := os.Stat(s.CredentialsFile); os.IsNotExist(err) {
		return errors.Wrap(err, "no key file")
	}

	// Read the file
	data, err := os.ReadFile(s.CredentialsFile)
	if err != nil {
		return errors.Wrap(err, "read key file")
	}
	s.Credential = string(data)

	return nil
}

func isValidURL(uri string) bool {
	if _, err := url.ParseRequestURI(uri); err != nil {
		return false
	}
	return true
}
