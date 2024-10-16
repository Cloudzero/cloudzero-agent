package handler

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestDeploymentHandler_Create(t *testing.T) {
	settings := &config.Settings{}
	db := storage.SetupDatabase()
	writer := storage.NewWriter(db)
	errChan := make(chan error)
	handler := NewDeploymentHandler(writer, settings, errChan)

	testDeploymentName := "test-deployment"
	testNamespace := "default"
	testLabels := map[string]string{
		"label-key": "label-value",
	}
	testAnnotations := map[string]string{
		"annotation-key": "annotation-value",
	}
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDeploymentName,
			Namespace:   testNamespace,
			Labels:      testLabels,
			Annotations: testAnnotations,
		},
	}

	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(v1.SchemeGroupVersion)
	raw, err := runtime.Encode(encoder, deployment)
	assert.NoError(t, err)

	request := &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}

	result, err := handler.Create(request)
	time.Sleep(2 * time.Second)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	var insertedRecord []storage.ResourceTags
	db.Find(&insertedRecord)
	// errs := <-errChan

	// assert.Equal(t, errs, nil) // todo: potentially handle after re-implementing concurrency
	assert.Len(t, insertedRecord, 1)
	assert.Equal(t, testDeploymentName, insertedRecord[0].Name)
	assert.Equal(t, &testNamespace, insertedRecord[0].Namespace)
	assert.Equal(t, &testLabels, insertedRecord[0].Labels)
	assert.Equal(t, &testAnnotations, insertedRecord[0].Annotations)
}
