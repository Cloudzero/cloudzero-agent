// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package streaming_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/microcosm-cc/bluemonday"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/backfiller"
	"github.com/cloudzero/cloudzero-agent/app/storage/streaming"
	"github.com/cloudzero/cloudzero-agent/app/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestStreamingStoreEndToEnd(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	log.Logger = zerolog.Nop()

	sink := newCollectorSink(t)
	defer sink.server.Close()

	settings := makeSettings(t, sink.server.URL)
	clock := &utils.Clock{}

	store := streaming.New(settings)
	wc, err := webhook.NewWebhookFactory(store, settings, clock)
	require.NoError(t, err)

	fakeClient := fake.NewClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "production",
				Labels: map[string]string{
					"environment": "prod",
					"team":        "platform",
					"unmatched":   "should-be-filtered",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "staging",
				Labels: map[string]string{
					"environment": "staging",
					"team":        "qa",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-1",
				Namespace: "production",
				Labels: map[string]string{
					"app":  "web",
					"team": "platform",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "c", Image: "nginx"}},
			},
		},
	)

	enum := backfiller.NewKubernetesObjectEnumerator(fakeClient, wc, settings)
	enum.DisableServiceWait()
	require.NoError(t, enum.Start(context.Background()))
	require.NoError(t, store.Flush())

	received := sink.timeseries()
	require.NotEmpty(t, received, "collector should have received timeseries")

	// Index received timeseries by __name__ for easier lookup.
	type ts struct {
		labels map[string]string
	}
	byName := map[string][]ts{}
	for _, series := range received {
		entry := ts{labels: map[string]string{}}
		var name string
		for _, l := range series.Labels {
			if l.Name == "__name__" {
				name = l.Value
			} else {
				entry.labels[l.Name] = l.Value
			}
		}
		byName[name] = append(byName[name], entry)
	}

	t.Logf("received %d timeseries across %d WriteRequests", len(received), sink.requestCount())
	for name, entries := range byName {
		t.Logf("  %s: %d entries", name, len(entries))
		for i, e := range entries {
			if i < 3 {
				t.Logf("    %v", e.labels)
			}
		}
	}

	// We should have namespace and pod label timeseries.
	nsLabels := byName["cloudzero_namespace_labels"]
	require.NotEmpty(t, nsLabels, "should have cloudzero_namespace_labels timeseries")

	// Find the "production" namespace entry.
	var prodEntry *ts
	for i := range nsLabels {
		if nsLabels[i].labels["namespace"] == "production" {
			prodEntry = &nsLabels[i]
			break
		}
	}
	require.NotNil(t, prodEntry, "should have namespace labels for 'production'")
	assert.Equal(t, "prod", prodEntry.labels["label_environment"])
	assert.Equal(t, "platform", prodEntry.labels["label_team"])
	assert.NotContains(t, prodEntry.labels, "label_unmatched",
		"unmatched label should be filtered by pattern")

	// Verify pod labels.
	podLabels := byName["cloudzero_pod_labels"]
	require.NotEmpty(t, podLabels, "should have cloudzero_pod_labels timeseries")
	var podEntry *ts
	for i := range podLabels {
		if podLabels[i].labels["pod"] == "web-1" {
			podEntry = &podLabels[i]
			break
		}
	}
	require.NotNil(t, podEntry, "should have pod labels for 'web-1'")
	assert.Equal(t, "web", podEntry.labels["label_app"])
	assert.Equal(t, "platform", podEntry.labels["label_team"])
}

// collectorSink is an httptest server that captures Prometheus WriteRequests.
type collectorSink struct {
	server *httptest.Server
	mu     sync.Mutex
	reqs   []prompb.WriteRequest
}

func newCollectorSink(t *testing.T) *collectorSink {
	t.Helper()
	s := &collectorSink{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/container-metrics", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("sink: read body: %v", err)
			http.Error(w, "read error", 500)
			return
		}
		r.Body.Close()

		decoded, err := snappy.Decode(nil, body)
		if err != nil {
			t.Errorf("sink: snappy decode: %v", err)
			http.Error(w, "decode error", 500)
			return
		}

		var wr prompb.WriteRequest
		if err := proto.Unmarshal(decoded, &wr); err != nil {
			t.Errorf("sink: proto unmarshal: %v", err)
			http.Error(w, "unmarshal error", 500)
			return
		}

		s.mu.Lock()
		s.reqs = append(s.reqs, wr)
		s.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	})
	s.server = httptest.NewServer(mux)
	return s
}

func (s *collectorSink) timeseries() []prompb.TimeSeries {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []prompb.TimeSeries
	for _, wr := range s.reqs {
		all = append(all, wr.Timeseries...)
	}
	return all
}

func (s *collectorSink) requestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reqs)
}

func makeSettings(t *testing.T, sinkURL string) *config.Settings {
	t.Helper()
	collectorURL := fmt.Sprintf("%s/v1/container-metrics?cloud_account_id=test&cluster_name=test&region=us-west-2", sinkURL)
	s := &config.Settings{
		CloudAccountID: "test",
		Region:         "us-west-2",
		ClusterName:    "test",
		Destination:    collectorURL,
		Logging:        config.Logging{Level: "error"},
		RemoteWrite: config.RemoteWrite{
			Host:            collectorURL,
			SendInterval:    1 * time.Second,
			MaxBytesPerSend: 500000,
			SendTimeout:     10 * time.Second,
			MaxRetries:      3,
		},
		K8sClient: config.K8sClient{PaginationLimit: 500},
		Database: config.Database{
			RetentionTime:   24 * time.Hour,
			CleanupInterval: 3 * time.Hour,
			BatchUpdateSize: 500,
		},
		Server: config.Server{
			Port:         0,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled:  true,
				Patterns: []string{`^environment$`, `^team$`, `^app$`},
				Resources: config.Resources{
					Namespaces: true,
					Pods:       true,
				},
			},
			Annotations: config.Annotations{
				Enabled:   false,
				Resources: config.Resources{},
			},
		},
	}
	s.Filters.Policy = *bluemonday.UGCPolicy()
	s.LabelMatches = compilePatterns(s.Filters.Labels.Patterns)
	s.AnnotationMatches = compilePatterns(s.Filters.Annotations.Patterns)
	return s
}

func compilePatterns(patterns []string) []regexp.Regexp {
	out := make([]regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			out = append(out, *re)
		}
	}
	return out
}

// Ensure we don't accidentally reference the unused runtime import from
// the fake clientset's k8sruntime alias.
var _ k8sruntime.Object = (*corev1.Pod)(nil)
