package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

type S3Config struct {
	AccessKey string `envconfig:"AWS_ACCESS_KEY_ID"`
	SecretKey string `envconfig:"AWS_SECRET_ACCESS_KEY"`
	Region    string `envconfig:"AWS_REGION"`
	Endpoint  string `envconfig:"AWS_ENDPOINT"`
}

func (c *S3Config) String() string {
	return fmt.Sprintf("AccessKey: %s, SecretKey: %s, Region: %s, Endpoint: %s", c.AccessKey, c.SecretKey, c.Region, c.Endpoint)
}

func (c *S3Config) Load() error {
	logrus.Info("loading s3 config")
	return envconfig.Process("awsconfig", c)
}
