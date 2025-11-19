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

# Save collection metadata
cat > "${OUTPUT_DIR}/metadata.txt" <<EOF
Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
Kubernetes Context: ${KUBECTX}
Namespace: ${KUBENS}
Collected by: $(whoami)@$(hostname)
Script Version: anaximander.sh
EOF

info "Gathering Helm release information..."
helm --kube-context "${KUBECTX}" -n "${KUBENS}" list > "${OUTPUT_DIR}/helm-list.txt" 2>&1 || {
    echo "Failed to get Helm releases (this may be expected if not using Helm)" > "${OUTPUT_DIR}/helm-list.txt"
}

info "Gathering resource listings..."
eval "${KUBECMD} get all" > "${OUTPUT_DIR}/get-all.txt" 2>&1

info "Gathering secrets list..."
eval "${KUBECMD} get secrets" > "${OUTPUT_DIR}/get-secrets.txt" 2>&1

info "Gathering resource descriptions..."
eval "${KUBECMD} describe all" > "${OUTPUT_DIR}/describe-all.txt" 2>&1

info "Calculating secret sizes..."
{
    echo "Secret Data Sizes (bytes in .data field)"
    echo "=========================================="
    echo ""

    # Get all secret names
    SECRET_NAMES=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get secrets -o jsonpath='{.items[*].metadata.name}')

    for SECRET in ${SECRET_NAMES}; do
        # Get the secret and calculate total bytes in .data
        SIZE=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get secret "${SECRET}" -o json | \
            jq '[.data | to_entries[] | .value | length] | add // 0')

        echo "${SECRET}: ${SIZE} bytes"
    done
} > "${OUTPUT_DIR}/secret-sizes.txt" 2>&1

info "Gathering pod logs..."
# Get all pods in the namespace
PODS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -o jsonpath='{.items[*].metadata.name}')

for POD in ${PODS}; do
    # Get container names for this pod
    CONTAINERS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pod "${POD}" -o jsonpath='{.spec.containers[*].name}')

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
            kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${POD}" -c "${CONTAINER}" 2>&1 || echo "Failed to get current logs"

            echo ""
            echo "--- Previous Logs (if available) ---"
            kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${POD}" -c "${CONTAINER}" --previous 2>&1 || echo "No previous logs available"
        } > "${OUTPUT_FILE}"
    done
done

info "Gathering job logs..."
# Get all jobs in the namespace
JOBS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get jobs -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

if [ -n "${JOBS}" ]; then
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
            JOB_PODS=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods --selector=job-name="${JOB}" -o jsonpath='{.items[*].metadata.name}')

            for JOB_POD in ${JOB_PODS}; do
                echo "--- Logs from pod ${JOB_POD} ---"
                kubectl --context "${KUBECTX}" -n "${KUBENS}" logs "${JOB_POD}" 2>&1 || echo "Failed to get logs for ${JOB_POD}"
                echo ""
            done
        } > "${OUTPUT_FILE}"
    done
else
    echo "No jobs found in namespace" > "${OUTPUT_DIR}/job-logs.txt"
fi

info "Gathering events..."
eval "${KUBECMD} get events --sort-by='.lastTimestamp'" > "${OUTPUT_DIR}/events.txt" 2>&1

info "Gathering ConfigMaps..."
eval "${KUBECMD} get configmaps" > "${OUTPUT_DIR}/get-configmaps.txt" 2>&1

info "Gathering network policies..."
eval "${KUBECMD} get networkpolicies -o yaml" > "${OUTPUT_DIR}/network-policies.yaml" 2>&1

info "Gathering pod resource usage (kubectl top)..."
eval "${KUBECMD} top pods" > "${OUTPUT_DIR}/top-pods.txt" 2>&1 || {
    echo "Failed to get pod resource usage (metrics-server may not be installed)" > "${OUTPUT_DIR}/top-pods.txt"
}

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

info "Gathering scrape configuration..."
{
    echo "==================================="
    echo "Scrape Configuration Collection"
    echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "==================================="
    echo ""

    # Find server pod (the one that runs Prometheus/Alloy)
    echo "Looking for CloudZero Agent server pod..."
    SERVER_POD=$(kubectl --context "${KUBECTX}" -n "${KUBENS}" get pods -l app.kubernetes.io/component=server -o jsonpath='{.items[0].metadata.name}' 2>&1)

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

info "Gathering cAdvisor metrics (from first node)..."
# Get first node name for configuration verification
NODE=$(kubectl --context "${KUBECTX}" get nodes -o jsonpath='{.items[0].metadata.name}' 2>&1)

if [ -n "${NODE}" ]; then
    info "  Collecting cAdvisor metrics from node ${NODE}..."
    OUTPUT_FILE="${OUTPUT_DIR}/cadvisor-metrics.txt"

    {
        echo "==================================="
        echo "Node: ${NODE}"
        echo "Collection Time: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "==================================="
        echo ""

        kubectl --context "${KUBECTX}" get --raw "/api/v1/nodes/${NODE}/proxy/metrics/cadvisor" 2>&1 || echo "Failed to get cAdvisor metrics for ${NODE}"
    } > "${OUTPUT_FILE}"
else
    echo "Failed to get node" > "${OUTPUT_DIR}/cadvisor-metrics.txt"
fi

info "Creating archive..."
ARCHIVE_NAME="${OUTPUT_DIR}.tar.gz"
tar -czf "${ARCHIVE_NAME}" "${OUTPUT_DIR}"

info "Diagnostic collection complete!"
echo ""
echo "Output directory: ${OUTPUT_DIR}"
echo "Archive created: ${ARCHIVE_NAME}"
echo ""
echo "Please provide the archive file (${ARCHIVE_NAME}) to CloudZero support."
echo "Note: Review the contents before sharing to ensure no sensitive information is included."
