package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/snappy"
)

func main() {
	http.HandleFunc("/v1/container-metrics", validateHandler)

	fmt.Println("Test server started at :8081")
	http.ListenAndServe(":8081", nil)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	if query.Get("cluster_name") == "" || query.Get("cloud_account_id") == "" || query.Get("region") == "" {
		http.Error(w, "Missing required query parameters", http.StatusBadRequest)
		return
	}

	compressedData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	data, err := snappy.Decode(nil, compressedData)
	if err != nil {
		http.Error(w, "Failed to uncompress data", http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join("/app/test-output", fmt.Sprintf("%d.json", time.Now().Unix()))
	err = os.WriteFile(filePath, data, 0o644)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data received and written to file"))
}
