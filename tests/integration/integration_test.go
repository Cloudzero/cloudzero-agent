package integration

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func generateRequest(method, url string, req Request) (*http.Request, error) {
	query := "?"
	for k, v := range req.QueryParams {
		query += fmt.Sprintf("%s=%s&", k, v)
	}
	query = query[:len(query)-1]

	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(bodyBytes); err != nil {
		return nil, fmt.Errorf("failed to compress body: %v", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	httpReq, err := http.NewRequest(method, url+query, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Encoding", "gzip")

	return httpReq, nil
}

func NewAdmissionRequest() []byte {
	// Create an AdmissionRequest
	admissionRequest := &v1.AdmissionRequest{
		UID: types.UID("12345"),
		Kind: metav1.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Pod",
		},
		Resource: metav1.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		},
		Name:      "example-pod",
		Namespace: "default",
		Operation: v1.Create,
		Object: runtime.RawExtension{
			Raw: []byte(`{
                "apiVersion": "v1",
                "kind": "Pod",
                "metadata": {
                    "name": "example-pod",
                    "namespace": "default"
                },
                "spec": {
                    "containers": [{
                        "name": "example-container",
                        "image": "example-image"
                    }]
                }
            }`),
		},
	}

	// Create an AdmissionReview
	admissionReview := &v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: admissionRequest,
	}

	// Marshal the AdmissionReview to JSON
	admissionReviewJSON, err := json.Marshal(admissionReview)
	if err != nil {
		fmt.Printf("Error marshaling AdmissionReview: %v\n", err)
		return nil
	}
	return admissionReviewJSON
}

type Request struct {
	Method      string
	QueryParams map[string]string
	Body        []byte
}

type BodyParams struct {
	Kind       string
	UID        string
	ObjectName string
}

func TestIntegrationInvalidResponses(t *testing.T) {
	tests := []struct {
		name           string
		requests       []Request
		method         string
		expectedStatus int
	}{
		{
			name:   "Invalid route request should return 404",
			method: http.MethodPost,
			requests: []Request{
				{QueryParams: map[string]string{}, Body: []byte{}},
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range tt.requests {
				log.Info().Msg("Running test")

				foo := NewAdmissionRequest()
				if foo == nil {
					t.Fatalf("Failed to create fake request")
				}
				// Use a value from the config package
				httpReq, err := generateRequest(req.Method, BaseURL, Request{Body: foo, QueryParams: req.QueryParams})

				if err != nil {
					t.Fatalf("Failed to generate request: %v", err)
				}

				resp, err := http.DefaultClient.Do(httpReq)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				}

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						t.Fatalf("Failed to read response body: %v", err)
					}
					t.Logf("Response body: %s", body)
				}
			}
		})
	}
}

// for _, req := range tt.requests {
// 	httpReq, err := generateRequest(tt.url, req)
// 	if err != nil {
// 		t.Fatalf("Failed to generate request: %v", err)
// 	}

// 	resp, err := http.DefaultClient.Do(httpReq)
// 	if err != nil {
// 		t.Fatalf("Failed to send request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != tt.expectedStatus {
// 		t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
// 	}

// 	if resp.StatusCode == http.StatusOK {
// 		body, err := ioutil.ReadAll(resp.Body)
// 		if err != nil {
// 			t.Fatalf("Failed to read response body: %v", err)
// 		}
// 		t.Logf("Response body: %s", body)
// 	}
// }

// func NewAdmissionRequestBody(params BodyParams) map[string]interface{} {
// 	// Set default values if not provided
// 	if params.Kind == "" {
// 		params.Kind = "AdmissionReview"
// 	}
// 	if params.UID == "" {
// 		params.UID = "12345"
// 	}
// 	if params.ObjectName == "" {
// 		params.ObjectName = "test-pod"
// 	}

// 	// Create the Body object
// 	body := map[string]interface{}{
// 		"kind": params.Kind,
// 		"request": map[string]interface{}{
// 			"uid": params.UID,
// 			"object": map[string]interface{}{
// 				"metadata": map[string]interface{}{
// 					"name": params.ObjectName,
// 				},
// 			},
// 		},
// 	}

//		return body
//	}
// func generateRequest(url string, req Request) (*http.Request, error) {
// 	query := "?"
// 	for k, v := range req.QueryParams {
// 		query += fmt.Sprintf("%s=%s&", k, v)
// 	}
// 	query = query[:len(query)-1]

// 	bodyBytes, err := json.Marshal(req.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal body: %v", err)
// 	}

// 	var buf bytes.Buffer
// 	gz := gzip.NewWriter(&buf)
// 	if _, err := gz.Write(bodyBytes); err != nil {
// 		return nil, fmt.Errorf("failed to compress body: %v", err)
// 	}
// 	if err := gz.Close(); err != nil {
// 		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
// 	}

// 	httpReq, err := http.NewRequest(http.MethodPost, url+query, &buf)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %v", err)
// 	}
// 	httpReq.Header.Set("Content-Encoding", "gzip")

// 	return httpReq, nil
// }

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %v", err)
// 	}
// 	httpReq.Header.Set("Content-Encoding", "gzip")

// 	return httpReq, nil
// }
