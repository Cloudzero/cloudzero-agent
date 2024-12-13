// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func makeNamespaceRequest(record TestRecord) *hook.Request {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(corev1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, namespace)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func TestAllHandlers_Create(t *testing.T) {
	settings := NewTestSettings()
	db := storage.SetupDatabase()
	writer := storage.NewWriter(db, settings)
	errChan := make(chan error)
	handler := NewNamespaceHandler(writer, settings, errChan)
	var testRecords []TestRecord
	testRecord := TestRecord{
		Type:         config.Namespace,
		Name:         "test-ns",
		MetricLabels: map[string]string{"namespace": "foobar"},
		Labels:       map[string]string{"label-key": "ns-label-value"},
	}
	request := makeNamespaceRequest(testRecord)
	result, err := handler.Create(request)
	testRecords = append(testRecords, testRecord)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	reader := storage.NewReader(db, settings)
	currentTime := time.Now().UTC()
	insertedRecords, _ := reader.ReadData(currentTime)

	for i, record := range insertedRecords {
		assert.Equal(t, testRecords[i].Name, record.Name)
		assert.Nil(t, testRecords[i].Namespace)
		assert.Equal(t, testRecords[i].Labels, *record.Labels)
		assert.Nil(t, testRecords[i].Annotations)
	}
	t.Cleanup(func() {
		dbCleanup(db)
	})
}
