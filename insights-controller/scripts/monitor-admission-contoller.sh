#!/usr/bin/env bash

set -euo pipefail

# Define the namespace
NAMESPACE="cloudzero-insights"

# Define a label selector to find the right pods, adjust this based on your deployment specifics
LABEL_SELECTOR="app=webhook-server"

# Get the latest pod based on the start time
POD_NAME=$(kubectl -n ${NAMESPACE} get pods -l ${LABEL_SELECTOR} --sort-by=.metadata.creationTimestamp -o jsonpath="{.items[-1].metadata.name}")

if [ -z "$POD_NAME" ]; then
    echo "No pods found with label ${LABEL_SELECTOR} in namespace ${NAMESPACE}"
    exit 1
else
    echo "Fetching logs from the latest pod: $POD_NAME"
    # Fetch logs
    kubectl -n ${NAMESPACE} logs -f ${POD_NAME} | jq
fi
