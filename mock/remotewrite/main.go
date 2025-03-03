package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

type remoteWrite struct {
	files map[string]*file
}

type file struct {
	refID   string
	url     string
	created time.Time
}

// Request and response structures
type uploadRequest struct {
	Files []struct {
		ReferenceID string `json:"reference_id"`
	} `json:"files"`
}

type uploadResponse map[string]string

type abandonRequest []struct {
	ReferenceID string `json:"reference_id"`
	Reason      string `json:"reason"`
}

type apiResponse struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

func newRemoteWrite() *remoteWrite {
	return &remoteWrite{
		files: make(map[string]*file),
	}
}

func authMiddleware(next http.Handler, apiKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")

		// Check if the header is present and matches the expected value
		if authHeader == "" {
			writeAPIResponse(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		if authHeader != apiKey {
			writeAPIResponse(w, http.StatusForbidden, "Invalid API key")
			return
		}

		// If authentication is successful, call the next handler
		next.ServeHTTP(w, r)
	})
}

func (rw *remoteWrite) Handler(apiKey string) http.Handler {
	mux := http.NewServeMux()

	// Cluster status endpoint
	mux.Handle("/cluster_status", authMiddleware(http.HandlerFunc(rw.status), apiKey))

	// Upload endpoint for pre-signed URLs
	mux.Handle("/upload", authMiddleware(http.HandlerFunc(rw.upload), apiKey))

	// Abandon endpoint
	mux.Handle("/abandon", authMiddleware(http.HandlerFunc(rw.abandon), apiKey))

	return mux
}

// Generate a random reference ID
func generateRefID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Helper to check required query parameters
func checkRequiredParams(r *http.Request, params ...string) error {
	for _, param := range params {
		if r.URL.Query().Get(param) == "" {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}

// Helper to write JSON response
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// Helper to write API response format
func writeAPIResponse(w http.ResponseWriter, statusCode int, message string) {
	response := apiResponse{
		StatusCode: statusCode,
		Body:       fmt.Sprintf("{\"message\": \"%s\"}", message),
	}
	writeJSONResponse(w, statusCode, response)
}

// Handle cluster status data upload
func (rw *remoteWrite) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check required query parameters
	err := checkRequiredParams(r, "cluster_name", "cloud_account_id")
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if there's a body
	if r.Body == nil {
		writeAPIResponse(w, http.StatusBadRequest, "No body in request")
		return
	}

	// Parse the JSON body
	var requestBody struct {
		Body string `json:"body"`
	}

	err = json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Unable to decode status data")
		return
	}

	// Mock successful response
	writeAPIResponse(w, http.StatusOK, "Cluster status accepted")
}

func (rw *remoteWrite) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check required query parameters
	err := checkRequiredParams(r, "cluster_name", "cloud_account_id", "region")
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse the count parameter if provided
	count := 1
	countParam := r.URL.Query().Get("count")
	if countParam != "" {
		count, err = strconv.Atoi(countParam)
		if err != nil || count <= 0 {
			writeAPIResponse(w, http.StatusBadRequest, "Invalid count parameter")
			return
		}
	}

	// Parse the request body
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// If files are provided in the request, use those reference IDs
	// Otherwise, generate the requested number of reference IDs
	var refIDs []string
	if len(req.Files) > 0 {
		for _, file := range req.Files {
			refIDs = append(refIDs, file.ReferenceID)
		}
	} else {
		for i := 0; i < count; i++ {
			refIDs = append(refIDs, generateRefID())
		}
	}

	// Generate response with pre-signed URLs
	response := make(uploadResponse)
	for _, refID := range refIDs {
		// Create a mock pre-signed URL
		url := fmt.Sprintf("https://example.com/upload/%s", refID)

		// Store the file reference
		rw.files[refID] = &file{
			refID:   refID,
			url:     url,
			created: time.Now(),
		}

		// Add to response
		response[refID] = url
	}

	// Set the X-CloudZero-Replay header
	w.Header().Set("X-CloudZero-Replay", "mock-replay-metadata")

	// Return the response
	writeJSONResponse(w, http.StatusOK, response)
}

func (rw *remoteWrite) abandon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check required query parameters
	err := checkRequiredParams(r, "cluster_name", "cloud_account_id", "region")
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse the request body
	var req abandonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Process each reference ID to be abandoned
	for _, item := range req {
		// Remove the file reference
		delete(rw.files, item.ReferenceID)
		// Note: In a real implementation, we might want to track abandoned files
		// or return errors for non-existent reference IDs
	}

	// Return success response
	writeAPIResponse(w, http.StatusOK, "Abandon request processed successfully")
}

func main() {
	// check for an api key
	apiKey, exists := os.LookupEnv("API_KEY")
	if !exists {
		apiKey = "ak-test"
	}

	// check for a port
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}

	rw := newRemoteWrite()
	http.Handle("/v1/container-metrics", rw.Handler(apiKey))
	fmt.Printf("Server is running on :%s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
