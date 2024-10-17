package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type MetricService struct {
	EnableSnowflakeTables bool `envconfig:"ENABLE_SNOWFLAKE" default:"false"`
}

func (c *MetricService) String() string {
	return fmt.Sprintf("MetricService{EnableSnowflakeTables: %t}", c.EnableSnowflakeTables)
}
func (c *MetricService) Load() error {
	return envconfig.Process("metricservice", c)
}
