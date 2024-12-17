// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
)

func TestFormatJobData(t *testing.T) {
	tests := []struct {
		name     string
		job      *batchv1.Job
		settings *config.Settings
		expected storage.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
				LabelMatches: []regexp.Regexp{
					*regexp.MustCompile("app"),
				},
				AnnotationMatches: []regexp.Regexp{
					*regexp.MustCompile("annotation-key"),
				},
			},
			expected: storage.ResourceTags{
				Type:      config.Job,
				Name:      "test-job",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"job":           "test-job",
					"namespace":     "default",
					"resource_type": "job",
				},
				Labels: &config.MetricLabelTags{
					"app": "test",
				},
				Annotations: &config.MetricLabelTags{
					"annotation-key": "annotation-value",
				},
			},
		},
		{
			name: "Test with labels and annotations disabled",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
				},
			},
			expected: storage.ResourceTags{
				Type:      config.Job,
				Name:      "test-job",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"job":           "test-job",
					"namespace":     "default",
					"resource_type": "job",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJobData(tt.job, tt.settings)
			if !reflect.DeepEqual(tt.expected.MetricLabels, tt.expected.MetricLabels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.MetricLabels, tt.expected.MetricLabels)
			}
			if !reflect.DeepEqual(tt.expected.Labels, tt.expected.Labels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Labels, tt.expected.Labels)
			}
			if !reflect.DeepEqual(tt.expected.Annotations, tt.expected.Annotations) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Annotations, tt.expected.Annotations)
			}
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Namespace, result.Namespace)
		})
	}
}

func TestNewJobHandler(t *testing.T) {
	tests := []struct {
		name     string
		writer   storage.DatabaseWriter
		settings *config.Settings
		errChan  chan<- error
	}{
		{
			name:   "Test with valid settings",
			writer: &mockDatabaseWriter{},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			errChan: make(chan error),
		},
		{
			name:     "Test with nil settings",
			writer:   &mockDatabaseWriter{},
			settings: nil,
			errChan:  make(chan error),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewJobHandler(tt.writer, tt.settings, tt.errChan)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.writer, handler.Writer)
			assert.Equal(t, tt.errChan, handler.ErrorChan)
		})
	}
}

func makeJobRequest(record TestRecord) *hook.Request {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	if record.Namespace != nil {
		job.Namespace = *record.Namespace
	}

	scheme := runtime.NewScheme()
	batchv1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(batchv1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, job)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func TestJobHandler_Create(t *testing.T) {
	tests := []struct {
		name     string
		settings *config.Settings
		request  *hook.Request
		expected *hook.Result
	}{
		{
			name: "Test create with labels and annotations enabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
				Namespace: stringPtr("default"),
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			expected: &hook.Result{Allowed: true},
		},
		{
			name: "Test create with labels and annotations disabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
				Namespace: stringPtr("default"),
			}),
			expected: &hook.Result{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &mockDatabaseWriter{}
			handler := NewJobHandler(writer, tt.settings, make(chan error))
			result, err := handler.Create(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			result, err = handler.Update(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
