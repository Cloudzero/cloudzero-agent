package types_test

import (
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/stretchr/testify/assert"
)

func TestNewAdmissionReviewV1Beta1_ValidRequest(t *testing.T) {
	input := &admissionv1beta1.AdmissionReview{
		Request: &admissionv1beta1.AdmissionRequest{
			UID:       "12345",
			Name:      "test-name",
			Namespace: "test-namespace",
			Operation: admissionv1beta1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    "test-group",
				Version:  "v1",
				Resource: "test-resource",
			},
			Kind: metav1.GroupVersionKind{
				Group:   "test-group",
				Version: "v1",
				Kind:    "test-kind",
			},
			UserInfo: authenticationv1.UserInfo{
				Username: "test-user",
			},
			DryRun: func(b bool) *bool { return &b }(true),
			Object: runtime.RawExtension{
				Raw: []byte(`{"key":"value"}`),
			},
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"oldKey":"oldValue"}`),
			},
		},
	}

	expected := types.AdmissionReview{
		OriginalAdmissionReview: input,
		ID:                      "12345",
		Name:                    "test-name",
		Namespace:               "test-namespace",
		Operation:               types.OperationCreate,
		Version:                 types.AdmissionReviewVersionV1beta1,
		RequestGVR: &metav1.GroupVersionResource{
			Group:    "test-group",
			Version:  "v1",
			Resource: "test-resource",
		},
		RequestGVK: &metav1.GroupVersionKind{
			Group:   "test-group",
			Version: "v1",
			Kind:    "test-kind",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: "test-user",
		},
		DryRun:       true,
		NewObjectRaw: []byte(`{"key":"value"}`),
		OldObjectRaw: []byte(`{"oldKey":"oldValue"}`),
	}

	result := types.NewAdmissionReviewV1Beta1(input)
	assert.Equal(t, expected, result)
}

func TestNewAdmissionReviewV1Beta1_NilDryRun(t *testing.T) {
	input := &admissionv1beta1.AdmissionReview{
		Request: &admissionv1beta1.AdmissionRequest{
			UID:       "67890",
			Name:      "test-name-2",
			Namespace: "test-namespace-2",
			Operation: admissionv1beta1.Update,
			Resource: metav1.GroupVersionResource{
				Group:    "test-group-2",
				Version:  "v1",
				Resource: "test-resource-2",
			},
			Kind: metav1.GroupVersionKind{
				Group:   "test-group-2",
				Version: "v1",
				Kind:    "test-kind-2",
			},
			UserInfo: authenticationv1.UserInfo{
				Username: "test-user-2",
			},
			DryRun: nil,
			Object: runtime.RawExtension{
				Raw: []byte(`{"key2":"value2"}`),
			},
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"oldKey2":"oldValue2"}`),
			},
		},
	}

	expected := types.AdmissionReview{
		OriginalAdmissionReview: input,
		ID:                      "67890",
		Name:                    "test-name-2",
		Namespace:               "test-namespace-2",
		Operation:               types.OperationUpdate,
		Version:                 types.AdmissionReviewVersionV1beta1,
		RequestGVR: &metav1.GroupVersionResource{
			Group:    "test-group-2",
			Version:  "v1",
			Resource: "test-resource-2",
		},
		RequestGVK: &metav1.GroupVersionKind{
			Group:   "test-group-2",
			Version: "v1",
			Kind:    "test-kind-2",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: "test-user-2",
		},
		DryRun:       false,
		NewObjectRaw: []byte(`{"key2":"value2"}`),
		OldObjectRaw: []byte(`{"oldKey2":"oldValue2"}`),
	}

	result := types.NewAdmissionReviewV1Beta1(input)
	assert.Equal(t, expected, result)
}

func TestNewAdmissionReviewV1_ValidRequest(t *testing.T) {
	input := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "abcde",
			Name:      "test-name-v1",
			Namespace: "test-namespace-v1",
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    "test-group-v1",
				Version:  "v1",
				Resource: "test-resource-v1",
			},
			Kind: metav1.GroupVersionKind{
				Group:   "test-group-v1",
				Version: "v1",
				Kind:    "test-kind-v1",
			},
			UserInfo: authenticationv1.UserInfo{
				Username: "test-user-v1",
			},
			DryRun: func(b bool) *bool { return &b }(true),
			Object: runtime.RawExtension{
				Raw: []byte(`{"key-v1":"value-v1"}`),
			},
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"oldKey-v1":"oldValue-v1"}`),
			},
		},
	}

	expected := types.AdmissionReview{
		OriginalAdmissionReview: input,
		ID:                      "abcde",
		Name:                    "test-name-v1",
		Namespace:               "test-namespace-v1",
		Operation:               types.OperationCreate,
		Version:                 types.AdmissionReviewVersionV1,
		RequestGVR: &metav1.GroupVersionResource{
			Group:    "test-group-v1",
			Version:  "v1",
			Resource: "test-resource-v1",
		},
		RequestGVK: &metav1.GroupVersionKind{
			Group:   "test-group-v1",
			Version: "v1",
			Kind:    "test-kind-v1",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: "test-user-v1",
		},
		DryRun:       true,
		NewObjectRaw: []byte(`{"key-v1":"value-v1"}`),
		OldObjectRaw: []byte(`{"oldKey-v1":"oldValue-v1"}`),
	}

	result := types.NewAdmissionReviewV1(input)
	assert.Equal(t, expected, result)
}

func TestNewAdmissionReviewV1_NilDryRun(t *testing.T) {
	input := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "fghij",
			Name:      "test-name-v1-2",
			Namespace: "test-namespace-v1-2",
			Operation: admissionv1.Update,
			Resource: metav1.GroupVersionResource{
				Group:    "test-group-v1-2",
				Version:  "v1",
				Resource: "test-resource-v1-2",
			},
			Kind: metav1.GroupVersionKind{
				Group:   "test-group-v1-2",
				Version: "v1",
				Kind:    "test-kind-v1-2",
			},
			UserInfo: authenticationv1.UserInfo{
				Username: "test-user-v1-2",
			},
			DryRun: nil,
			Object: runtime.RawExtension{
				Raw: []byte(`{"key-v1-2":"value-v1-2"}`),
			},
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"oldKey-v1-2":"oldValue-v1-2"}`),
			},
		},
	}

	expected := types.AdmissionReview{
		OriginalAdmissionReview: input,
		ID:                      "fghij",
		Name:                    "test-name-v1-2",
		Namespace:               "test-namespace-v1-2",
		Operation:               types.OperationUpdate,
		Version:                 types.AdmissionReviewVersionV1,
		RequestGVR: &metav1.GroupVersionResource{
			Group:    "test-group-v1-2",
			Version:  "v1",
			Resource: "test-resource-v1-2",
		},
		RequestGVK: &metav1.GroupVersionKind{
			Group:   "test-group-v1-2",
			Version: "v1",
			Kind:    "test-kind-v1-2",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: "test-user-v1-2",
		},
		DryRun:       false,
		NewObjectRaw: []byte(`{"key-v1-2":"value-v1-2"}`),
		OldObjectRaw: []byte(`{"oldKey-v1-2":"oldValue-v1-2"}`),
	}

	result := types.NewAdmissionReviewV1(input)
	assert.Equal(t, expected, result)
}
