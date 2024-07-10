package config

import (
	"fmt"
	"slices"

	"github.com/pkg/errors"
)

type Prometheus struct {
	KubeStateMetricsServiceEndpoint       string   `yaml:"kube_state_metrics_service_endpoint" env:"KMS_EP_URL" required:"true" env-description:"Kube State Metrics Service Endpoint"`
	PrometheusNodeExporterServiceEndpoint string   `yaml:"prometheus_node_exporter_service_endpoint" env:"NODE_EXPORTER_EP_URL" required:"true" env-description:"Prometheus Node Exporter Service Endpoint"`
	Configurations                        []string `yaml:"configurations"`
}

func (s *Prometheus) Validate() error {
	if s.KubeStateMetricsServiceEndpoint == "" {
		return errors.New(ErrNoKubeStateMetricsServiceEndpointMsg)
	}
	if !isValidURL(s.KubeStateMetricsServiceEndpoint) {
		return fmt.Errorf("invalid %s", s.KubeStateMetricsServiceEndpoint)
	}

	if s.PrometheusNodeExporterServiceEndpoint == "" {
		return fmt.Errorf(ErrNoPrometheusNodeExporterServiceEndpointMsg)
	}
	if !isValidURL(s.PrometheusNodeExporterServiceEndpoint) {
		return fmt.Errorf("URL format invalid: %s", s.PrometheusNodeExporterServiceEndpoint)
	}

	if len(s.Configurations) == 0 {
		s.Configurations = []string{
			"/etc/prometheus/prometheus.yml",
			"/etc/config/prometheus/configmaps/prometheus.yml",
		}
	} else {
		cleanedPaths := []string{}
		for _, location := range s.Configurations {
			if location == "" {
				continue
			}
			location, err := absFilePath(location)
			if err != nil {
				return err
			}
			if slices.Contains(cleanedPaths, location) {
				continue
			}
			cleanedPaths = append(cleanedPaths, location)
		}
		s.Configurations = cleanedPaths
	}
	return nil
}
