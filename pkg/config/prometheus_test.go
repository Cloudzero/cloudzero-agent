package config_test

import (
    "os"
    "testing"

    "github.com/cloudzero/cloudzero-agent-validator/pkg/config"
    "github.com/pkg/errors"
    "github.com/stretchr/testify/assert"
)

func TestPrometheus_Validate(t *testing.T) {
    wd, err := os.Getwd()
    assert.NoError(t, err)
    scrapeConfigFile := wd + "/testdata/prometheus.yml"
    tests := []struct {
        name     string
        prom     config.Prometheus
        expected error
    }{
        {
            name: "ValidPrometheus",
            prom: config.Prometheus{
                KubeStateMetricsServiceEndpoint: kmsServiceEndpoint,
                Configurations:                  []string{scrapeConfigFile},
            },
            expected: nil,
        },
        {
            name: "MissingKubeStateMetricsServiceEndpoint",
            prom: config.Prometheus{
                Configurations: []string{scrapeConfigFile},
            },
            expected: errors.New(config.ErrNoKubeStateMetricsServiceEndpointMsg),
        },
        {
            name: "MissingScrapeConfigLocation",
            prom: config.Prometheus{
                KubeStateMetricsServiceEndpoint: kmsServiceEndpoint,
            },
            expected: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.prom.Validate()
            if tt.expected == nil {
                assert.NoError(t, err)
                return
            }
            assert.Equal(t, tt.expected.Error(), err.Error())
        })
    }
}
