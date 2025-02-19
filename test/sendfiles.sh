#!/bin/bash

# ===================================================================
# Script Name: upload_snappy_files.sh
# Description: Uploads all .snappy files in the current directory
#              to a specified API endpoint using curl.
# Author: Your Name
# Date: YYYY-MM-DD
# ===================================================================

# ----------------------------
# Configuration Variables
# ----------------------------

# Cluster configuration
CLUSTER_NAME="test-stack"
REGION="us-east-1"
CLOUD_ACCOUNT_ID="1234567890"

# API Endpoint
API_ENDPOINT="http://localhost:8080/metrics"

# HTTP Headers
CONTENT_TYPE="application/x-protobuf"
CONTENT_ENCODING="snappy"
# ----------------------------
# Function: Upload File
# ----------------------------

upload_file() {
    local file_path="$1"

    # Construct the full URL with query parameters
    local url="${API_ENDPOINT}?region=${REGION}&cloud_account_id=${CLOUD_ACCOUNT_ID}&cluster_name=${CLUSTER_NAME}"

    # Perform the POST request using curl
    response=$(curl -s -w "%{http_code}" -o /dev/null \
        -X POST "$url" \
        -H "Content-Type: ${CONTENT_TYPE}" \
        -H "Content-Encoding: ${CONTENT_ENCODING}" \
        --data-binary "@${file_path}")

    # Check the HTTP status code
    if [[ "$response" -eq 200 || "$response" -eq 201 || "$response" -eq 204 ]]; then
        echo "[SUCCESS] Uploaded '${file_path}' successfully. HTTP Status: ${response}"
    else
        echo "[ERROR] Failed to upload '${file_path}'. HTTP Status: ${response}"
    fi
}

# ----------------------------
# Main Script Execution
# ----------------------------

# Check if there are any .snappy files in the current directory
shopt -s nullglob
snappy_files=(files/*.snappy)
shopt -u nullglob

if [[ ${#snappy_files[@]} -eq 0 ]]; then
    echo "[INFO] No .snappy files found in the current directory."
    exit 0
fi

echo "Starting upload of ${#snappy_files[@]} .snappy files to S3..."

# Iterate over each .snappy file and upload
for snappy_file in "${snappy_files[@]}"; do
    # Check if it's a regular file
    if [[ -f "$snappy_file" ]]; then
        upload_file "$snappy_file"
    else
        echo "[WARNING] '${snappy_file}' is not a regular file. Skipping."
    fi
done

echo "Upload process completed."
