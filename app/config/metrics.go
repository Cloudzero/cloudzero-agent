package config

import (
	"github.com/kelseyhightower/envconfig"
)

type MetricServiceConfig struct {
}

func (c *MetricServiceConfig) String() string {
	return ""
}
func (c *MetricServiceConfig) Load() error {
	return envconfig.Process("metricservice", c)
}
