package integration

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
)

func TestIntegrationValidResponses(t *testing.T) {
	tests := []struct {
		name           string
		requests       []Request
		route          string
		expectedStatus int
	}{
		{
			name: "Valid route request should return 200",
			requests: []Request{
				{QueryParams: map[string]string{
					"cluster_name":     ValidClusterName,
					"cloud_account_id": ValidAccountID,
					"region":           ValidRegion,
				}, Body: []byte{},
					Route:  PodRoute,
					Method: http.MethodPost,
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range tt.requests {
				log.Info().Msg("Running test")

				admissionReq := NewAdmissionRequest()
				if admissionReq == nil {
					t.Fatalf("Failed to create test admission request")
				}
				httpReq, err := generateRequest(req.Method, req.Route, BaseURL, Request{Body: admissionReq, QueryParams: req.QueryParams})

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
				time.Sleep(20 * time.Second)
			}
		})
	}
	t.Cleanup(func() {
		files, err := os.ReadDir(TestOutputDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}
		for _, file := range files {
			err := os.Remove(fmt.Sprintf("%s/%s", TestOutputDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to remove test file: %v", err)
			}
		}
	})
}

func TestIntegrationInvalidResponses(t *testing.T) {
	tests := []struct {
		name           string
		requests       []Request
		method         string
		route          string
		expectedStatus int
	}{
		{
			name:   "Invalid route request should return 404",
			method: http.MethodPost,
			requests: []Request{
				{QueryParams: map[string]string{}, Body: []byte{},
					Route: "/invalid-route",
				},
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
				httpReq, err := generateRequest(req.Method, req.Route, BaseURL, Request{Body: foo, QueryParams: req.QueryParams})

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
