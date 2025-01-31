//coverage:ignore
package types

import (
	"time"

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
	Id             string            `json:"id"         parquet:"-"`
	ClusterName    string            `json:"cluster_name" parquet:"cluster_name"`
	CloudAccountID string            `json:"cloud_account_id" parquet:"cloud_account_id"`
	OrganizationID string            `json:"organization_id" parquet:"organization_id"`
	Year           string            `json:"year" parquet:"year"`
	Month          string            `json:"month" parquet:"month"`
	Day            string            `json:"day" parquet:"day"`
	Hour           string            `json:"hour" parquet:"hour"`
	Name           string            `json:"name"       parquet:"name"`
	CreatedAt      int64             `json:"created_at" parquet:"create_at,timestamp(microsecond)"`
	TimeStamp      int64             `json:"timestamp"  parquet:"timestamp,timestamp(microsecond)"`
	Labels         map[string]string `json:"labels"     parquet:"labels"`
	Value          string            `json:"value"      parquet:"value"`
}

func NewMetric(orgID, cloudAccountID, clusterName, name string, timeStamp int64, labels map[string]string, value string) Metric {
	if labels == nil {
		labels = make(map[string]string)
	}
	createAt := timestamp.Milli()
	t := time.Unix(0, timeStamp*int64(time.Millisecond))
	year := GetYear(t)
	month := GetMonth(t)
	day := GetDay(t)
	hour := GetHour(t)
	return Metric{
		Id:             uuid.New().String(),
		OrganizationID: orgID,
		CloudAccountID: cloudAccountID,
		ClusterName:    clusterName,
		Name:           name,
		CreatedAt:      createAt,
		Year:           year,
		Month:          month,
		Day:            day,
		Hour:           hour,
		TimeStamp:      timeStamp,
		Labels:         labels,
		Value:          value,
	}
}

type MetricRange struct {
	Metrics []Metric `json:"metrics"`
	Next    *string  `json:"next,omitempty"`
}

// GetYear extracts the year as a string with four digits.
func GetYear(t time.Time) string {
	return t.Format("2006")
}

// GetMonth extracts the month as a string with two digits (leading zero if necessary).
func GetMonth(t time.Time) string {
	return t.Format("01")
}

// GetDay extracts the day as a string with two digits (leading zero if necessary).
func GetDay(t time.Time) string {
	return t.Format("02")
}

// GetHour extracts the hour as a string with two digits (leading zero if necessary) in 24-hour format.
func GetHour(t time.Time) string {
	return t.Format("15")
}
