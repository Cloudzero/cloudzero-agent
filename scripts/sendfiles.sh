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
CLUSTER_NAME="rancher-stack"
REGION="us-east-1"
CLOUD_ACCOUNT_ID="1234567890"

# API Endpoint
API_ENDPOINT="https://ft0hrt94f8.execute-api.us-east-1.amazonaws.com/Stage/metrics"

# HTTP Headers
ORGANIZATION_ID="3e0e0d25-12345-12345-12345"
CONTENT_TYPE="application/x-protobuf"

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
        -H "organization_id: ${ORGANIZATION_ID}" \
        -H "Content-Type: ${CONTENT_TYPE}" \
        --data-binary "@${file_path}")

    # Check the HTTP status code
    if [[ "$response" -eq 200 || "$response" -eq 201 ]]; then
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
snappy_files=(*.snappy)
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
