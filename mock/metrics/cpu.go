package metrics

import (
	"fmt"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/google/uuid"
)

func GenerateCPUUsageRecords(
	input *MetricRecordInput,
	numCPUs int,
	utilizationRatio float64,
	samplesPerHour int,
	nodeName, namespace, podName string,
) []types.Metric {
	startTime := input.StartTime.UTC()
	endTime := input.EndTime.UTC()

	var records []types.Metric

	// Calculate total CPU seconds available per hour.
	secondsPerHour := float64(numCPUs) * 3600 * utilizationRatio

	// CPU seconds added incrementally per sample.
	incrementPerSample := secondsPerHour / float64(samplesPerHour)

	// Calculate the interval between samples.
	// Since samplesPerHour gives the samples in one hour (60 minutes), we calculate
	// the duration of each interval. time.Minute is an int64 nanosecond duration,
	// so converting via float64 is safe for our purposes.
	sampleInterval := time.Duration((60.0 / float64(samplesPerHour)) * float64(time.Minute))

	// Make sure we work in UTC so that the time formatting
	// (for example, including the "+0000" offset) is consistent.
	currentTime := startTime.UTC()
	currentCPUUsage := 0.0

	// While currentTime is before or equal to endTime.
	for currentTime.Before(endTime) {
		// Generate samples for each hour.
		for sampleIndex := 0; sampleIndex < samplesPerHour; sampleIndex++ {
			sampleTime := currentTime.Add(time.Duration(sampleIndex) * sampleInterval)
			if sampleTime.After(endTime) {
				break
			}

			// Increment the cumulative CPU usage.
			currentCPUUsage += incrementPerSample

			// You might want to mimic the Python timestamp formatting:
			// Python: 'YYYY-MM-DD HH:MM:SS.mmm +0000'
			// Go format layout: "2006-01-02 15:04:05.000 -0700"
			// Here we also use sampleTime to fill the CreatedAt and TimeStamp fields.
			timestampStr := sampleTime.Format("2006-01-02 15:04:05.000 -0700")
			_ = timestampStr // the string is only shown in Python comments

			labels := map[string]string{
				"__name__":  "container_cpu_usage_seconds_total",
				"node":      nodeName,
				"namespace": namespace,
				"pod":       podName,
			}

			records = append(records, types.Metric{
				ID:             uuid.New(),
				ClusterName:    input.ClusterName,
				CloudAccountID: input.CloudAccountID,
				MetricName:     "container_cpu_usage_seconds_total",
				NodeName:       nodeName,
				CreatedAt:      sampleTime,
				TimeStamp:      sampleTime,
				Labels:         labels,
				Value:          fmt.Sprintf("%.3f", currentCPUUsage),
			})
		}

		// Move to the next hour.
		currentTime = currentTime.Add(time.Hour)
	}

	return records
}
