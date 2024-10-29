package handler

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func dbCleanup(db *gorm.DB) {
	instance, _ := db.DB()
	instance.Close()
}

type TestRecord struct {
	Type         config.ResourceType
	Name         string
	Namespace    *string
	MetricLabels map[string]string
	Labels       map[string]string
	Annotations  map[string]string
}

func makeDeploymentRequest(record TestRecord) *hook.Request {

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Namespace:   *record.Namespace,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(v1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, deployment)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func NewTestSettings() *config.Settings {
	filter := config.Filters{Labels: config.Labels{Enabled: true, Patterns: []string{"*"}}, Annotations: config.Annotations{Enabled: true, Patterns: []string{"*"}}}
	compiledPatterns := []regexp.Regexp{}
	testPattern, _ := regexp.Compile(".*")
	compiledPatterns = append(compiledPatterns, *testPattern)
	return &config.Settings{RemoteWrite: config.RemoteWrite{MaxBytesPerSend: 10000, SendInterval: 3}, Filters: filter, LabelMatches: compiledPatterns, AnnotationMatches: compiledPatterns}
}

func TestDeploymentHandler_Create(t *testing.T) {
	settings := NewTestSettings()
	db := storage.SetupDatabase()
	writer := storage.NewWriter(db)
	errChan := make(chan error)
	handler := NewDeploymentHandler(writer, settings, errChan)
	var testRecords []TestRecord
	for i := 1; i <= 20; i++ {
		namespace := fmt.Sprintf("ns-%d", i)
		testRecord := TestRecord{
			Type:         config.Deployment,
			Name:         fmt.Sprintf("test-deployment-%d", i),
			Namespace:    &namespace,
			MetricLabels: map[string]string{"workload": fmt.Sprintf("test-deployment-%d", i)},
			Labels:       map[string]string{"label-key": fmt.Sprintf("label-value-%d", i)},
			Annotations: map[string]string{
				"annotation-key": fmt.Sprintf("annotation-value-%d", i),
			},
		}
		request := makeDeploymentRequest(testRecord)
		result, err := handler.Create(request)
		testRecords = append(testRecords, testRecord)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	reader := storage.NewReader(db, settings)
	currentTime := time.Now().UTC()
	insertedRecords, _ := reader.ReadData(currentTime)

	assert.Len(t, insertedRecords, 20)
	for i, record := range insertedRecords {
		assert.Equal(t, testRecords[i].Name, record.Name)
		assert.Equal(t, *testRecords[i].Namespace, *record.Namespace)
		assert.Equal(t, testRecords[i].Labels, *record.Labels)
		assert.Equal(t, testRecords[i].Annotations, *record.Annotations)
	}
	t.Cleanup(func() {
		dbCleanup(db)
	})
}
