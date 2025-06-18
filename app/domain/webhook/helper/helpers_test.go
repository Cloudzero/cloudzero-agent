// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package helper_test contains tests
package helper_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type msi = map[string]interface{}

// Deployment.
var (
	rawYAMLDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  namespace: "test-ns"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
`

	rawJSONDeployment = `
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "nginx-test",
    "namespace": "test-ns"
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "nginx"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "nginx"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "nginx",
            "image": "nginx"
          }
        ]
      }
    }
  }
}
`

	k8sObjDeployment = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-test",
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &([]int32{3}[0]),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": "nginx",
			}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "nginx", Image: "nginx"},
					},
				},
			},
		},
	}

	unstructuredObjDeployment = &unstructured.Unstructured{
		Object: msi{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": msi{
				"name":      "nginx-test",
				"namespace": "test-ns",
			},

			"spec": msi{
				"replicas": int64(3),
				"selector": msi{
					"matchLabels": msi{
						"app": "nginx",
					},
				},
				"template": msi{
					"metadata": msi{
						"labels": msi{
							"app": "nginx",
						},
					},
					"spec": msi{
						"containers": []interface{}{
							msi{
								"name":  "nginx",
								"image": "nginx",
							},
						},
					},
				},
			},
		},
	}
)

// PodExecOptions (special K8s runtime.Object types that don't satisfy metav1.Object).
var (
	rawJSONPodExecOptions = `
{
   "kind":"PodExecOptions",
   "apiVersion":"v1",
   "stdin":true,
   "stdout":true,
   "tty":true,
   "container":"nginx",
   "command":[
      "/bin/sh"
   ]
}`
	UnstructuredObjPodExecOptions = &unstructured.Unstructured{
		Object: msi{
			"apiVersion": "v1",
			"kind":       "PodExecOptions",
			"stdin":      true,
			"stdout":     true,
			"tty":        true,
			"container":  "nginx",
			"command": []interface{}{
				"/bin/sh",
			},
		},
	}
)

func TestObjectCreator(t *testing.T) {
	tests := map[string]struct {
		objectCreator func() types.ObjectCreator
		raw           string
		expObj        types.K8sObject
		expErr        bool
	}{
		"Static with invalid objects should fail.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    "{",
			expErr: true,
		},

		"Static object creation with JSON raw data should return the object on the specific type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    rawJSONDeployment,
			expObj: k8sObjDeployment,
		},

		"Static object creation with YAML raw data should return the object on the specific type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewStaticObjectCreator(&appsv1.Deployment{})
			},
			raw:    rawYAMLDeployment,
			expObj: k8sObjDeployment,
		},

		"Static with unstructured object creation should return the object on unstructured type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewStaticObjectCreator(&unstructured.Unstructured{})
			},
			raw:    rawYAMLDeployment,
			expObj: unstructuredObjDeployment,
		},

		"Dynamic with invalid objects should fail.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewDynamicObjectCreator()
			},
			raw:    "{",
			expErr: true,
		},

		"Dynamic object creation with JSON should return the object on an inferred type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewDynamicObjectCreator()
			},
			raw:    rawJSONDeployment,
			expObj: k8sObjDeployment,
		},

		"Dynamic object creation with YAML should return the object on an inferred type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewDynamicObjectCreator()
			},
			raw:    rawYAMLDeployment,
			expObj: k8sObjDeployment,
		},

		"Static with unstructured using only runtime.Object compatible creation should return the object unstructured type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewStaticObjectCreator(&unstructured.Unstructured{})
			},
			raw:    rawJSONPodExecOptions,
			expObj: UnstructuredObjPodExecOptions,
		},

		"Dynamic only runtime.Object creation should return the object on an inferred unstructured type.": {
			objectCreator: func() types.ObjectCreator {
				return helper.NewDynamicObjectCreator()
			},
			raw:    rawJSONPodExecOptions,
			expObj: UnstructuredObjPodExecOptions,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			oc := test.objectCreator()
			gotObj, err := oc.NewObject([]byte(test.raw))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expObj, gotObj)
			}
		})
	}
}
