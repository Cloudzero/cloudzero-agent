package http

import (
	"bytes"
	"fmt"

	"github.com/rs/zerolog/log"

	"net/http"
	"os"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	v1 "k8s.io/api/apps/v1"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
)

func collectMetrics(dp v1.Deployment) []prompb.TimeSeries {
	var timeSeries []prompb.TimeSeries
	ts := prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__name__",
				Value: "kube_pod_labels",
			},
		},
		Samples: []prompb.Sample{
			{
				Value:     1,
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
			},
		},
	}

	for labelKey, labelValue := range dp.Spec.Template.ObjectMeta.Labels {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  fmt.Sprintf("label_%s", labelKey),
			Value: labelValue,
		})
	}
	timeSeries = append(timeSeries, ts)

	return timeSeries
}

func pushMetrics(remoteWriteURL string, apiKey string, timeSeries []prompb.TimeSeries) error {

	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(writeRequest)
	if err != nil {
		return fmt.Errorf("error marshaling WriteRequest: %v", err)
	}

	compressed := snappy.Encode(nil, data)

	req, err := http.NewRequest("POST", remoteWriteURL, bytes.NewBuffer(compressed))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	return nil
}

func PushLabels(dp v1.Deployment, settings *config.Settings) {
	apiKey, err := os.ReadFile(settings.APIKeyPath)

	if err != nil {
		log.Err(err).Msg("Failed to read API key")
	}
	remoteWriteURL := fmt.Sprintf("%s/v1/container-metrics?cluster_name=%s&cloud_account_id=%s&region=%s", settings.Host, settings.ClusterName, settings.CloudAccountID, settings.Region)
	timeSeries := collectMetrics(dp)
	err = pushMetrics(remoteWriteURL, string(apiKey), timeSeries)
	if err != nil {
		log.Err(err).Msg("Failed to push metrics")
	}

}
