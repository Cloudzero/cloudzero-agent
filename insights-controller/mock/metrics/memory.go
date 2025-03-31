package metrics

import (
	"fmt"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/google/uuid"
)

func GenerateMemoryUsageRecords(
	input *MetricRecordInput,
	totalMemoryBytesAvailable int64,
	maxPercentage float64,
	samplesPerHour int,
	nodeName, namespace, podName string,
) []types.Metric {
	startTime := input.StartTime.UTC()
	endTime := input.EndTime.UTC()

	var records []types.Metric

	// Calculate the maximum bytes available per hour times maxPercentage.
	maxBytesPerHour := float64(totalMemoryBytesAvailable) * maxPercentage

	// Determine the incremental increase per sample.
	incrementPerSample := maxBytesPerHour / float64(samplesPerHour)

	// Calculate the interval between samples.
	sampleInterval := time.Duration((60.0 / float64(samplesPerHour)) * float64(time.Minute))

	// Ensure we work in UTC.
	currentTime := startTime.UTC()

	// Iterate over each hour until currentTime is after endTime.
	for currentTime.Before(endTime) {
		// Generate samples for this hour.
		for sampleIndex := 0; sampleIndex < samplesPerHour; sampleIndex++ {
			sampleTime := currentTime.Add(time.Duration(sampleIndex) * sampleInterval)
			if sampleTime.After(endTime) {
				break
			}

			// Compute memory usage for this sample.
			memoryUsage := incrementPerSample * float64(sampleIndex+1)

			// Prepare the labels based on the original Python dictionary.
			labels := map[string]string{
				"__name__":  "container_memory_working_set_bytes",
				"node":      nodeName,
				"namespace": namespace,
				"pod":       podName,
			}

			records = append(records, types.Metric{
				ID:             uuid.New(),
				ClusterName:    input.ClusterName,
				CloudAccountID: input.CloudAccountID,
				MetricName:     "container_memory_working_set_bytes",
				NodeName:       nodeName,
				CreatedAt:      sampleTime,
				TimeStamp:      sampleTime,
				Labels:         labels,
				Value:          fmt.Sprintf("%.3f", memoryUsage),
			})
		}

		// Move to the next hour.
		currentTime = currentTime.Add(time.Hour)
	}

	return records
}
