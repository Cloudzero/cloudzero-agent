package domain_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/domain/filter"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

var defaultTestMetric types.Metric = types.Metric{
	ID:             uuid.MustParse("d64271ef-46af-4ef9-94b6-c537a186b01d"),
	ClusterName:    "aws-cirrus-brahms",
	CloudAccountID: "8675309",
	MetricName:     "container_network_transmit_bytes_total",
	NodeName:       "ip-192-168-62-22.ec2.internal",
	CreatedAt:      time.UnixMilli(1740671645978).UTC(),
	TimeStamp:      time.UnixMilli(1740671634889).UTC(),
	Labels: map[string]string{
		"image":                     "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause:3.5",
		"instance":                  "ip-192-168-62-22.ec2.internal",
		"k8s_io_cloud_provider_aws": "eb707f9bdba15de05a26c5a3b4a909ee",
		"name":                      "340166e10e91263f42abc91459ed3523ced66250f87df0f945a5816dea321452",
		"namespace":                 "kube-system",
		"pod":                       "kube-proxy-9bnjh",
	},
	Value: "990",
}

func TestMetricFilter_Filter(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.Metrics
		metrics       []types.Metric
		cost          []types.Metric
		observability []types.Metric
		dropped       []types.Metric
		wantErr       bool
	}{
		{
			name: "no-match",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "not_container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "not_container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost:          nil,
			observability: nil,
			dropped: []types.Metric{
				defaultTestMetric,
			},
		},
		{
			name: "cost-match",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "not_container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: []types.Metric{
				defaultTestMetric,
			},
			observability: nil,
			dropped:       nil,
		},
		{
			name: "observability-match",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "not_container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: nil,
			observability: []types.Metric{
				defaultTestMetric,
			},
			dropped: nil,
		},
		{
			name: "both-match",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: []types.Metric{
				defaultTestMetric,
			},
			observability: []types.Metric{
				defaultTestMetric,
			},
			dropped: nil,
		},
		{
			name: "filter-label",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				CostLabels: []filter.FilterEntry{
					{
						Pattern: "image",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				ObservabilityLabels: []filter.FilterEntry{
					{
						Pattern: "po",
						Match:   filter.FilterMatchTypePrefix,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: []types.Metric{
				func() types.Metric {
					m := defaultTestMetric
					m.Labels = map[string]string{
						"image": "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause:3.5",
					}
					return m
				}(),
			},
			observability: []types.Metric{
				func() types.Metric {
					m := defaultTestMetric
					m.Labels = map[string]string{
						"pod": "kube-proxy-9bnjh",
					}
					return m
				}(),
			},
			dropped: nil,
		},
		{
			name: "default-allow",
			cfg:  config.Metrics{},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: []types.Metric{
				defaultTestMetric,
			},
			observability: []types.Metric{
				defaultTestMetric,
			},
			dropped: nil,
		},
		{
			name: "empty-allow",
			cfg: config.Metrics{
				Cost:                []filter.FilterEntry{},
				Observability:       []filter.FilterEntry{},
				CostLabels:          []filter.FilterEntry{},
				ObservabilityLabels: []filter.FilterEntry{},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost: []types.Metric{
				defaultTestMetric,
			},
			observability: []types.Metric{
				defaultTestMetric,
			},
			dropped: nil,
		},
		{
			name: "multiple-metrics-mixed-results",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "container_network_transmit_bytes_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "container_cpu_usage_seconds_total",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric, // matches cost
				func() types.Metric {
					m := defaultTestMetric
					m.ID = uuid.MustParse("a64271ef-46af-4ef9-94b6-c537a186b01d")
					m.MetricName = "container_cpu_usage_seconds_total"
					return m
				}(), // matches observability
				func() types.Metric {
					m := defaultTestMetric
					m.ID = uuid.MustParse("b64271ef-46af-4ef9-94b6-c537a186b01d")
					m.MetricName = "container_memory_usage_bytes"
					return m
				}(), // matches neither - should be dropped
			},
			cost: []types.Metric{
				defaultTestMetric,
			},
			observability: []types.Metric{
				func() types.Metric {
					m := defaultTestMetric
					m.ID = uuid.MustParse("a64271ef-46af-4ef9-94b6-c537a186b01d")
					m.MetricName = "container_cpu_usage_seconds_total"
					return m
				}(),
			},
			dropped: []types.Metric{
				func() types.Metric {
					m := defaultTestMetric
					m.ID = uuid.MustParse("b64271ef-46af-4ef9-94b6-c537a186b01d")
					m.MetricName = "container_memory_usage_bytes"
					return m
				}(),
			},
		},
		{
			name: "all-dropped",
			cfg: config.Metrics{
				Cost: []filter.FilterEntry{
					{
						Pattern: "non_existent_metric",
						Match:   filter.FilterMatchTypeExact,
					},
				},
				Observability: []filter.FilterEntry{
					{
						Pattern: "another_non_existent_metric",
						Match:   filter.FilterMatchTypeExact,
					},
				},
			},
			metrics: []types.Metric{
				defaultTestMetric,
			},
			cost:          nil,
			observability: nil,
			dropped: []types.Metric{
				defaultTestMetric,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mf, err := domain.NewMetricFilter(&tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("MetricFilter.Filter() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			costMetrics, observabilityMetrics, droppedMetrics := mf.Filter(tt.metrics)

			if diff := cmp.Diff(costMetrics, tt.cost); diff != "" {
				t.Errorf("MetricFilter.Filter() cost mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(observabilityMetrics, tt.observability); diff != "" {
				t.Errorf("MetricFilter.Filter() observability mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(droppedMetrics, tt.dropped); diff != "" {
				t.Errorf("MetricFilter.Filter() dropped mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
