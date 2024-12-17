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

func TestFormatCronData(t *testing.T) {
	tests := []struct {
		name     string
		cronjob  *batchv1.CronJob
		settings *config.Settings
		expected storage.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			cronjob: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cronjob",
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
							CronJobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							CronJobs: true,
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
				Type:      config.CronJob,
				Name:      "test-cronjob",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"cronjob":       "test-cronjob",
					"namespace":     "default",
					"resource_type": "cronjob",
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
			cronjob: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cronjob",
					Namespace: "default",
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							CronJobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							CronJobs: false,
						},
					},
				},
			},
			expected: storage.ResourceTags{
				Type:      config.CronJob,
				Name:      "test-cronjob",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"namespace":     "default",
					"resource_type": "cronjob",
					"cronjob":       "test-cronjob",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCronJobData(tt.cronjob, tt.settings)
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

func TestNewCronJobHandler(t *testing.T) {
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
							CronJobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							CronJobs: true,
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
			handler := NewCronJobHandler(tt.writer, tt.settings, tt.errChan)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.writer, handler.Writer)
			assert.Equal(t, tt.errChan, handler.ErrorChan)
		})
	}
}

func makeCronJobRequest(record TestRecord) *hook.Request {
	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	if record.Namespace != nil {
		cronjob.Namespace = *record.Namespace
	}

	scheme := runtime.NewScheme()
	batchv1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(batchv1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, cronjob)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func TestCronJobHandler_Create(t *testing.T) {
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
							CronJobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							CronJobs: true,
						},
					},
				},
			},
			request: makeCronJobRequest(TestRecord{
				Name:      "test-cronjob",
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
							CronJobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							CronJobs: false,
						},
					},
				},
			},
			request: makeCronJobRequest(TestRecord{
				Name:      "test-cronjob",
				Namespace: stringPtr("default"),
			}),
			expected: &hook.Result{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &mockDatabaseWriter{}
			handler := NewCronJobHandler(writer, tt.settings, make(chan error))
			result, err := handler.Create(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			result, err = handler.Update(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
