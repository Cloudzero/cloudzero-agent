package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
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
	// if query.Get("cluster_name") == "" || query.Get("cloud_account_id") == "" || query.Get("region") == "" {
	if query.Get("cluster_name") == "" || query.Get("cloud_account_id") == "" || query.Get("region") == "" {
		// if query.Get("cluster_name") == "" {
		errString := fmt.Sprintf("Missing required query parameters: cluster_name=%s, cloud_account_id=%s, region=%s", query.Get("cluster_name"), query.Get("cloud_account_id"), query.Get("region"))
		http.Error(w, errString, http.StatusBadRequest)
		return
	}

	compressedData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// data, err := snappy.Decode(nil, compressedData)
	// if err != nil {
	// 	http.Error(w, "Failed to uncompress data", http.StatusInternalServerError)
	// 	return
	// }

	filePath := filepath.Join("/app/test-output", filepath.Base(r.URL.Path)+".json")
	err = ioutil.WriteFile(filePath, compressedData, 0644)
	// err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data received and written to file"))
}
