//coverage:ignore
package types

import (
	"github.com/go-obvious/timestamp"
	"github.com/google/uuid"
)

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Sample struct {
	Value     *float64 `json:"value"`
	Timestamp string   `json:"timestamp"`
}

type TimeSeries struct {
	Labels  []Label  `json:"labels"`
	Samples []Sample `json:"samples"`
}

type InputData struct {
	TimeSeries []TimeSeries `json:"timeseries"`
}

type Metric struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	CreatedAt int64             `json:"created_at"`
	TimeStamp int64             `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
	Value     string            `json:"value"`
}

func NewMetric(name string, timeStamp int64, labels map[string]string, value string) Metric {
	return Metric{
		Id:        uuid.New().String(),
		Name:      name,
		CreatedAt: timestamp.Milli(),
		TimeStamp: timeStamp,
		Labels:    labels,
		Value:     value,
	}
}

type MetricRange struct {
	Metrics []Metric `json:"metrics"`
	Next    *string  `json:"next,omitempty"`
}
