#!/usr/bin/env bash

set -euo pipefail

# Anaximander - CloudZero Agent Diagnostic Information Gatherer
#
# This script gathers comprehensive diagnostic information from a Kubernetes cluster
# to help CloudZero support troubleshoot customer issues.

function usage() {
    cat <<EOF
Usage: $(basename "$0") <kube-context> <namespace> [output-dir]

Gathers diagnostic information from a Kubernetes cluster for CloudZero support.

Arguments:
  kube-context    Kubernetes context to use (see 'kubectl config get-contexts' for available contexts)
  namespace       Namespace where CloudZero Agent is installed
  output-dir      Optional output directory (default: cloudzero-diagnostics-<timestamp>)

Example:
  $(basename "$0") my-cluster cloudzero-agent
  $(basename "$0") prod-cluster cloudzero /tmp/diagnostics

The script will create a directory containing:
  - Helm release information
  - Kubernetes resource listings and descriptions
  - Secret size information
  - Container logs from all pods
  - Job logs
  - Network policies
  - Pod resource usage (kubectl top)
  - Service mesh detection (Istio, Linkerd, Consul)
  - Scrape configuration
  - cAdvisor metrics

EOF
    exit 1
}

function error() {
    echo "Error: $*" >&2
    exit 1
}

function info() {
    echo "==> $*"
}

# Run a diagnostic command and track results
# Usage: run_diagnostic "description" "output_file" "command..."
function run_diagnostic() {
    local description="$1"
    local output_file="$2"
    shift 2
    local cmd="$*"

    if eval "$cmd" > "${output_file}" 2>&1; then
        echo "[SUCCESS] ${description}" >> "${COMMAND_RESULTS_FILE}"
        return 0
    else
        local exit_code=$?
        echo "[FAILED]  ${description} (exit code: ${exit_code})" >> "${COMMAND_RESULTS_FILE}"
        # Prepend failure notice to output file
        local temp_file
        temp_file=$(mktemp)
        {
            echo "[FAILED] Command failed with exit code ${exit_code}"
            echo "Command: ${cmd}"
            echo "=========================================="
            echo ""
            cat "${output_file}"
        } > "${temp_file}"
        mv "${temp_file}" "${output_file}"
        FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
        return 0  # Don't fail the script
    fi
}

# Parse arguments
if [ $# -lt 2 ] || [ $# -gt 3 ]; then
    usage
fi

KUBECTX="$1"
KUBENS="$2"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
OUTPUT_DIR="${3:-cloudzero-diagnostics-${TIMESTAMP}}"

# Set up kubectl command
KUBECMD="kubectl --context \"${KUBECTX}\" -n \"${KUBENS}\""

# Verify context exists
if ! kubectl config get-contexts "${KUBECTX}" &>/dev/null; then
    error "Kubernetes context '${KUBECTX}' not found"
fi

# Verify namespace exists
if ! kubectl --context "${KUBECTX}" get namespace "${KUBENS}" &>/dev/null; then
    error "Namespace '${KUBENS}' not found in context '${KUBECTX}'"
fi

# Create output directory
mkdir -p "${OUTPUT_DIR}"
info "Creating diagnostic bundle in: ${OUTPUT_DIR}"

# Initialize command tracking
COMMAND_RESULTS_FILE="${OUTPUT_DIR}/command-results.txt"
FAILED_COMMANDS=0

# Initialize results file
{
    echo "Command Results Summary"
    echo "======================="
    echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo ""
} > "${COMMAND_RESULTS_FILE}"

# Save collection metadata
cat > "${OUTPUT_DIR}/metadata.txt" <<EOF
Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
Kubernetes Context: ${KUBECTX}
Namespace: ${KUBENS}
Collected by: $(whoami)@$(hostname)
Script Version: anaximander.sh
EOF

info "Gathering Helm release information..."
run_diagnostic "helm list" "${OUTPUT_DIR}/helm-list.txt" \
    "helm --kube-context '${KUBECTX}' -n '${KUBENS}' list"

info "Gathering resource listings..."
run_diagnostic "kubectl get all" "${OUTPUT_DIR}/get-all.txt" "${KUBECMD} get all"

info "Gathering secrets list..."
run_diagnostic "kubectl get secrets" "${OUTPUT_DIR}/get-secrets.txt" "${KUBECMD} get secrets"

info "Gathering resource descriptions..."
run_diagnostic "kubectl describe all" "${OUTPUT_DIR}/describe-all.txt" "${KUBECMD} describe all"

info "Calculating secret sizes..."
{
    echo "Secret Data Sizes (bytes in .data field)"
    echo "=========================================="
    echo ""

    # Get all secret names
    SECRET_NAMES=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get secrets -o jsonpath='{.items[*].metadata.name}')

    if [ -z "${SECRET_NAMES}" ]; then
        echo "No secrets found or failed to list secrets"
        echo "[FAILED]  Calculate secret sizes (no secrets found)" >> "${COMMAND_RESULTS_FILE}"
        FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
    else
        for SECRET in ${SECRET_NAMES}; do
            # Get the secret and calculate total bytes in .data
            SIZE=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get secret "${SECRET}" -o json | \
                jq '[.data | to_entries[] | .value | length] | add // 0')

            echo "${SECRET}: ${SIZE} bytes"
        done
        echo "[SUCCESS] Calculate secret sizes" >> "${COMMAND_RESULTS_FILE}"
    fi
} > "${OUTPUT_DIR}/secret-sizes.txt" 2>&1

info "Gathering pod logs..."
# Get all pods in the namespace
PODS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

if [ -z "${PODS}" ]; then
    echo "No pods found in namespace" > "${OUTPUT_DIR}/pod-logs-summary.txt"
    echo "[FAILED]  Gather pod logs (no pods found)" >> "${COMMAND_RESULTS_FILE}"
    FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
else
    POD_LOG_FAILURES=0
    for POD in ${PODS}; do
        # Get container names for this pod
        CONTAINERS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pod "${POD}" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")

        for CONTAINER in ${CONTAINERS}; do
            info "  Collecting logs for ${POD}/${CONTAINER}..."
            OUTPUT_FILE="${OUTPUT_DIR}/${POD}-${CONTAINER}-logs.txt"

            {
                echo "==================================="
                echo "Pod: ${POD}"
                echo "Container: ${CONTAINER}"
                echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
                echo "==================================="
                echo ""

                # Get current logs
                echo "--- Current Logs ---"
                if ! kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${POD}" -c "${CONTAINER}" 2>&1; then
                    echo "Failed to get current logs"
                    POD_LOG_FAILURES=$((POD_LOG_FAILURES + 1))
                fi

                echo ""
                echo "--- Previous Logs (if available) ---"
                kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${POD}" -c "${CONTAINER}" --previous 2>&1 || echo "No previous logs available"
            } > "${OUTPUT_FILE}"
        done
    done

    if [ "${POD_LOG_FAILURES}" -gt 0 ]; then
        echo "[FAILED]  Gather pod logs (${POD_LOG_FAILURES} container(s) failed)" >> "${COMMAND_RESULTS_FILE}"
        FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
    else
        echo "[SUCCESS] Gather pod logs" >> "${COMMAND_RESULTS_FILE}"
    fi
fi

info "Gathering job logs..."
# Get all jobs in the namespace
JOBS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get jobs -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

if [ -n "${JOBS}" ]; then
    JOB_LOG_FAILURES=0
    for JOB in ${JOBS}; do
        info "  Collecting logs for job ${JOB}..."
        OUTPUT_FILE="${OUTPUT_DIR}/job-${JOB}-logs.txt"

        {
            echo "==================================="
            echo "Job: ${JOB}"
            echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
            echo "==================================="
            echo ""

            # Get the pods for this job
            JOB_PODS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods --selector=job-name="${JOB}" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

            if [ -z "${JOB_PODS}" ]; then
                echo "No pods found for job ${JOB}"
                JOB_LOG_FAILURES=$((JOB_LOG_FAILURES + 1))
            else
                for JOB_POD in ${JOB_PODS}; do
                    echo "--- Logs from pod ${JOB_POD} ---"
                    if ! kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${JOB_POD}" 2>&1; then
                        echo "Failed to get logs for ${JOB_POD}"
                        JOB_LOG_FAILURES=$((JOB_LOG_FAILURES + 1))
                    fi
                    echo ""
                done
            fi
        } > "${OUTPUT_FILE}"
    done

    if [ "${JOB_LOG_FAILURES}" -gt 0 ]; then
        echo "[FAILED]  Gather job logs (${JOB_LOG_FAILURES} job(s) failed)" >> "${COMMAND_RESULTS_FILE}"
        FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
    else
        echo "[SUCCESS] Gather job logs" >> "${COMMAND_RESULTS_FILE}"
    fi
else
    echo "No jobs found in namespace" > "${OUTPUT_DIR}/job-logs.txt"
    echo "[SUCCESS] Gather job logs (no jobs in namespace)" >> "${COMMAND_RESULTS_FILE}"
fi

info "Gathering events..."
run_diagnostic "kubectl get events" "${OUTPUT_DIR}/events.txt" "${KUBECMD} get events --sort-by='.lastTimestamp'"

info "Gathering ConfigMaps..."
run_diagnostic "kubectl get configmaps" "${OUTPUT_DIR}/get-configmaps.txt" "${KUBECMD} get configmaps"

info "Gathering network policies..."
run_diagnostic "kubectl get networkpolicies" "${OUTPUT_DIR}/network-policies.yaml" "${KUBECMD} get networkpolicies -o yaml"

info "Gathering pod resource usage (kubectl top)..."
run_diagnostic "kubectl top pods" "${OUTPUT_DIR}/top-pods.txt" "${KUBECMD} top pods"

info "Detecting service mesh configuration..."
{
    echo "==================================="
    echo "Service Mesh Detection"
    echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "==================================="
    echo ""

    # Check for Istio
    echo "--- Istio Detection ---"
    echo ""
    echo "Istio namespace:"
    kubectl --context "${KUBECTX}" get namespace istio-system 2>&1 || echo "  istio-system namespace not found"
    echo ""

    echo "Istio CRDs:"
    kubectl --context "${KUBECTX}" get crd 2>&1 | grep -E "istio\.io|networking\.istio\.io" || echo "  No Istio CRDs found"
    echo ""

    echo "Namespace istio-injection label:"
    kubectl --context "${KUBECTX}" get namespace "${KUBENS}" -o jsonpath='{.metadata.labels.istio-injection}' 2>&1 || echo "  No istio-injection label"
    echo ""
    echo ""

    echo "Pods with Istio sidecar in namespace ${KUBENS}:"
    PODS_WITH_ISTIO=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -o json 2>&1 | jq -r '.items[] | select(.spec.containers[]?.name == "istio-proxy") | .metadata.name' || echo "")
    if [ -n "${PODS_WITH_ISTIO}" ]; then
        echo "${PODS_WITH_ISTIO}"
    else
        echo "  No pods with istio-proxy container found"
    fi
    echo ""

    echo "Istio resources in namespace ${KUBENS}:"
    kubectl --context "${KUBECTX}" -n "${KUBENS}" get virtualservices,destinationrules,gateways,serviceentries,sidecars,peerauthentications 2>&1 || echo "  No Istio resources found"
    echo ""
    echo ""

    # Check for Linkerd
    echo "--- Linkerd Detection ---"
    echo ""
    echo "Linkerd namespace:"
    kubectl --context "${KUBECTX}" get namespace linkerd 2>&1 || echo "  linkerd namespace not found"
    echo ""

    echo "Linkerd CRDs:"
    kubectl --context "${KUBECTX}" get crd 2>&1 | grep "linkerd.io" || echo "  No Linkerd CRDs found"
    echo ""

    echo "Pods with Linkerd sidecar in namespace ${KUBENS}:"
    PODS_WITH_LINKERD=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -o json 2>&1 | jq -r '.items[] | select(.spec.containers[]?.name == "linkerd-proxy") | .metadata.name' || echo "")
    if [ -n "${PODS_WITH_LINKERD}" ]; then
        echo "${PODS_WITH_LINKERD}"
    else
        echo "  No pods with linkerd-proxy container found"
    fi
    echo ""
    echo ""

    # Check for Consul
    echo "--- Consul Service Mesh Detection ---"
    echo ""
    echo "Consul namespace:"
    kubectl --context "${KUBECTX}" get namespace consul 2>&1 || echo "  consul namespace not found"
    echo ""

    echo "Consul CRDs:"
    kubectl --context "${KUBECTX}" get crd 2>&1 | grep "consul.hashicorp.com" || echo "  No Consul CRDs found"
    echo ""

    echo "Pods with Consul sidecar in namespace ${KUBENS}:"
    PODS_WITH_CONSUL=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -o json 2>&1 | jq -r '.items[] | select(.spec.containers[]?.name == "consul-connect-envoy-sidecar" or .spec.containers[]?.name == "consul-dataplane") | .metadata.name' || echo "")
    if [ -n "${PODS_WITH_CONSUL}" ]; then
        echo "${PODS_WITH_CONSUL}"
    else
        echo "  No pods with Consul sidecar found"
    fi
    echo ""

} > "${OUTPUT_DIR}/service-mesh-detection.txt" 2>&1
echo "[SUCCESS] Detect service mesh configuration" >> "${COMMAND_RESULTS_FILE}"

info "Gathering scrape configuration..."
{
    echo "==================================="
    echo "Scrape Configuration Collection"
    echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "==================================="
    echo ""

    # Find server pod (the one that runs Prometheus/Alloy)
    echo "Looking for CloudZero Agent server pod..."
    SERVER_POD=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -l app.kubernetes.io/part-of=cloudzero-agent,app.kubernetes.io/name=server -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

    if [ -z "${SERVER_POD}" ]; then
        echo "No server pod found"
    else
        echo "Found server pod: ${SERVER_POD}"
        echo ""

        # Get all ConfigMap volume mounts from the server pod
        echo "Examining ConfigMap mounts in server pod..."

        # Get the pod spec as JSON for easier parsing
        POD_SPEC=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pod "${SERVER_POD}" -o json 2>&1)

        if [ -z "${POD_SPEC}" ]; then
            echo "Could not retrieve pod spec"
        else
            # Look for ConfigMap volumes and their mount paths
            # We're looking for the one mounted at /etc/config/prometheus/configmaps/ or /etc/alloy
            CONFIG_CM=""
            CONFIG_MODE=""

            # Check for Prometheus mode mount (/etc/config/prometheus/configmaps/)
            PROM_CM=$(echo "${POD_SPEC}" | jq -r '
                .spec.volumes[] |
                select(.configMap != null) |
                select(.name == "config-volume") |
                .configMap.name
            ' 2>/dev/null)

            if [ -n "${PROM_CM}" ] && [ "${PROM_CM}" != "null" ]; then
                # Verify it's mounted at the prometheus path
                PROM_MOUNT=$(echo "${POD_SPEC}" | jq -r '
                    .spec.containers[].volumeMounts[]? |
                    select(.name == "config-volume" and (.mountPath | contains("prometheus"))) |
                    .mountPath
                ' 2>/dev/null | head -1)

                if [ -n "${PROM_MOUNT}" ] && [ "${PROM_MOUNT}" != "null" ]; then
                    CONFIG_CM="${PROM_CM}"
                    CONFIG_MODE="prometheus"
                    echo "Found Prometheus mode configuration"
                    echo "ConfigMap: ${CONFIG_CM}"
                    echo "Mount path: ${PROM_MOUNT}"
                fi
            fi

            # Check for Alloy mode mount (/etc/alloy) if Prometheus not found
            if [ -z "${CONFIG_CM}" ]; then
                ALLOY_CM=$(echo "${POD_SPEC}" | jq -r '
                    .spec.volumes[] |
                    select(.configMap != null) |
                    select(.name == "config-volume") |
                    .configMap.name
                ' 2>/dev/null)

                if [ -n "${ALLOY_CM}" ] && [ "${ALLOY_CM}" != "null" ]; then
                    ALLOY_MOUNT=$(echo "${POD_SPEC}" | jq -r '
                        .spec.containers[].volumeMounts[]? |
                        select(.name == "config-volume" and (.mountPath | contains("alloy"))) |
                        .mountPath
                    ' 2>/dev/null | head -1)

                    if [ -n "${ALLOY_MOUNT}" ] && [ "${ALLOY_MOUNT}" != "null" ]; then
                        CONFIG_CM="${ALLOY_CM}"
                        CONFIG_MODE="alloy"
                        echo "Found Alloy mode configuration"
                        echo "ConfigMap: ${CONFIG_CM}"
                        echo "Mount path: ${ALLOY_MOUNT}"
                    fi
                fi
            fi

            echo ""

            if [ -z "${CONFIG_CM}" ]; then
                echo "Could not identify configuration ConfigMap from pod mounts"
            else
                # Dump the entire scrape configuration
                echo "Saving scrape configuration from ${CONFIG_CM}..."
                echo ""

                if [ "${CONFIG_MODE}" = "prometheus" ]; then
                    echo "Extracting prometheus.yml..."
                    kubectl --context "${KUBECTX}" -n "${KUBENS}" get configmap "${CONFIG_CM}" -o jsonpath='{.data.prometheus\.yml}' > "${OUTPUT_DIR}/prometheus-config.yml" 2>&1

                    if [ -s "${OUTPUT_DIR}/prometheus-config.yml" ]; then
                        echo "Saved complete Prometheus configuration to prometheus-config.yml"
                    else
                        echo "Could not extract prometheus.yml from ConfigMap ${CONFIG_CM}" > "${OUTPUT_DIR}/prometheus-config.yml"
                    fi

                elif [ "${CONFIG_MODE}" = "alloy" ]; then
                    echo "Extracting alloy-config.river..."
                    kubectl --context "${KUBECTX}" -n "${KUBENS}" get configmap "${CONFIG_CM}" -o jsonpath='{.data.alloy-config\.river}' > "${OUTPUT_DIR}/alloy-config.river" 2>&1

                    if [ -s "${OUTPUT_DIR}/alloy-config.river" ]; then
                        echo "Saved complete Alloy configuration to alloy-config.river"
                    else
                        echo "Could not extract alloy-config.river from ConfigMap ${CONFIG_CM}" > "${OUTPUT_DIR}/alloy-config.river"
                    fi
                fi
            fi
        fi
    fi
    echo ""

} > "${OUTPUT_DIR}/scrape-config-info.txt" 2>&1
echo "[SUCCESS] Gather scrape configuration" >> "${COMMAND_RESULTS_FILE}"

info "Gathering cAdvisor metrics (from first node)..."
# Get first node name for configuration verification
NODE=$(kubectl --context "${KUBECTX}" get nodes -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -n "${NODE}" ]; then
    info "  Collecting cAdvisor metrics from node ${NODE}..."

    # Create final output file with header
    {
        echo "==================================="
        echo "Node: ${NODE}"
        echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "==================================="
        echo ""
    } > "${OUTPUT_DIR}/cadvisor-metrics.txt"

    # Collect metrics and append (track via run_diagnostic)
    CADVISOR_TEMP=$(mktemp)
    run_diagnostic "Get cAdvisor metrics from node ${NODE}" "${CADVISOR_TEMP}" \
        "kubectl --context '${KUBECTX}' get --raw '/api/v1/nodes/${NODE}/proxy/metrics/cadvisor'"
    cat "${CADVISOR_TEMP}" >> "${OUTPUT_DIR}/cadvisor-metrics.txt"
    rm -f "${CADVISOR_TEMP}"
else
    echo "[FAILED] Could not determine node name - cAdvisor metrics not collected" > "${OUTPUT_DIR}/cadvisor-metrics.txt"
    echo "[FAILED]  Get node name for cAdvisor metrics" >> "${COMMAND_RESULTS_FILE}"
    FAILED_COMMANDS=$((FAILED_COMMANDS + 1))
fi

# Finalize command results summary
{
    echo ""
    echo "======================="
    echo "Total failed: ${FAILED_COMMANDS}"
} >> "${COMMAND_RESULTS_FILE}"

info "Creating archive..."
ARCHIVE_NAME="${OUTPUT_DIR}.tar.gz"
tar -czf "${ARCHIVE_NAME}" "${OUTPUT_DIR}"

info "Diagnostic collection complete!"
echo ""
echo "Output directory: ${OUTPUT_DIR}"
echo "Archive created: ${ARCHIVE_NAME}"

if [ "${FAILED_COMMANDS}" -gt 0 ]; then
    echo ""
    echo "WARNING: ${FAILED_COMMANDS} diagnostic command(s) failed. See command-results.txt for details."
fi

echo ""
echo "Please provide the archive file (${ARCHIVE_NAME}) to CloudZero support."
echo "Note: Review the contents before sharing to ensure no sensitive information is included."
