#!/usr/bin/env sh
#
# kube-list-labels-annotations.sh
#
# List all labels and/or annotations from all resources in a Kubernetes cluster,
# sorted by frequency with counts.
#
# This script queries ALL resource types in the cluster (pods, services,
# deployments, configmaps, secrets, nodes, etc.) and aggregates their labels
# and/or annotations to show which ones are most commonly used.
#
# USAGE
#   ./scripts/kube-list-labels-annotations.sh [options] [-- kubectl-args...]
#
# OPTIONS
#   -l, --labels       Show labels (default if neither -l nor -a specified)
#   -a, --annotations  Show annotations
#   -v, --values       Show values for each key (hierarchical output)
#   -h, --help         Show this help message
#
#   Any arguments after '--' are passed directly to kubectl. This is useful
#   for specifying context, namespace restrictions, or authentication options.
#
# OUTPUT FORMAT
#   Without -v (keys only):
#      COUNT LABEL_KEY
#
#   With -v (keys and values):
#      COUNT LABEL_KEY
#            COUNT VALUE
#            COUNT VALUE
#            ...
#
#   Results are sorted by frequency (most common first) at both the key and
#   value levels. Counts are right-aligned with a minimum width of 6 characters.
#
# EXAMPLES
#   # List all label keys by frequency (default)
#   ./scripts/kube-list-labels-annotations.sh
#
#   # List all label keys with their values
#   ./scripts/kube-list-labels-annotations.sh --values
#   ./scripts/kube-list-labels-annotations.sh -v
#
#   # List annotation keys only
#   ./scripts/kube-list-labels-annotations.sh --annotations
#   ./scripts/kube-list-labels-annotations.sh -a
#
#   # List both labels and annotations with values
#   ./scripts/kube-list-labels-annotations.sh -l -a -v
#
#   # Use a specific kubectl context
#   ./scripts/kube-list-labels-annotations.sh -- --context my-cluster
#
#   # Combine with grep to find specific patterns
#   ./scripts/kube-list-labels-annotations.sh -v | grep -A 100 "app.kubernetes.io/name"
#
# REQUIREMENTS
#   - kubectl (configured with appropriate cluster access)
#   - jq (for JSON processing)
#   - Standard Unix tools: awk, sort, uniq, tr, sed
#
# NOTES
#   - This script queries all resource types, which can be slow on large clusters
#   - Permission errors for inaccessible resources are silently suppressed
#   - Results include both namespaced and cluster-scoped resources
#   - Empty labels/annotations objects are handled gracefully
#
# SEE ALSO
#   kubectl api-resources    - List available resource types
#   kubectl get              - Get resources
#   kubectl label            - Update labels on resources
#   kubectl annotate         - Update annotations on resources

set -eu

# Check dependencies
check_dep() {
    command -v "$1" >/dev/null 2>&1 || { echo "Error: $1 is required but not found" >&2; exit 1; }
}

check_dep kubectl
check_dep jq
check_dep awk
check_dep sort
check_dep uniq
check_dep tr
check_dep sed

show_labels=false
show_annotations=false
show_values=false
kubectl_args=""

usage() {
    sed -n '2,/^set -eu$/p' "$0" | sed -e 's/^#//' -e 's/^ //' -e '/^set -eu$/d'
    exit 0
}

while [ $# -gt 0 ]; do
    case $1 in
        -l|--labels)
            show_labels=true
            shift
            ;;
        -a|--annotations)
            show_annotations=true
            shift
            ;;
        -v|--values)
            show_values=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        --)
            shift
            kubectl_args="$*"
            break
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Use -h for help" >&2
            exit 1
            ;;
    esac
done

# Default to labels if neither specified
if [ "$show_labels" = "false" ] && [ "$show_annotations" = "false" ]; then
    show_labels=true
fi

# Get all resource types that support list
get_resource_types() {
    # shellcheck disable=SC2086
    kubectl $kubectl_args api-resources --verbs=list -o name 2>/dev/null | tr '\n' ',' | sed 's/,$//'
}

# Fetch all resources as JSON
fetch_resources() {
    resources=$(get_resource_types)
    # shellcheck disable=SC2086
    kubectl $kubectl_args get "$resources" -A -o json 2>/dev/null
}

# Extract keys only, sorted by frequency
extract_keys_by_frequency() {
    field=$1
    jq -r ".items[].metadata.${field} // {} | keys[]" | sort | uniq -c | sort -rn | awk '{printf "%6d %s\n", $1, $2}'
}

# Extract keys and values, both sorted by frequency
extract_keys_values_by_frequency() {
    field=$1
    jq -r ".items[].metadata.${field} // {} | to_entries[] | \"\(.key)\t\(.value)\"" | \
    sort | uniq -c | awk -F'\t' '
    {
        split($1, arr, " "); count = arr[1]; key = arr[2]; value = $2
        key_total[key] += count
        val_count[key, value] = count
        if (!(key in seen)) { keys[++n] = key; seen[key] = 1 }
        if (!((key, value) in val_seen)) { key_vals[key] = key_vals[key] "\t" value; val_seen[key, value] = 1 }
    }
    END {
        for (i = 1; i <= n; i++) for (j = i + 1; j <= n; j++) if (key_total[keys[j]] > key_total[keys[i]]) { t = keys[i]; keys[i] = keys[j]; keys[j] = t }
        for (i = 1; i <= n; i++) {
            k = keys[i]; printf "%6d %s\n", key_total[k], k
            m = split(key_vals[k], v, "\t")
            for (p = 2; p <= m; p++) for (q = p + 1; q <= m; q++) if (val_count[k, v[q]] > val_count[k, v[p]]) { t = v[p]; v[p] = v[q]; v[q] = t }
            for (p = 2; p <= m; p++) printf "        %6d %s\n", val_count[k, v[p]], v[p]
        }
    }'
}

# Main
tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

fetch_resources > "$tmpfile"

if [ "$show_labels" = "true" ]; then
    echo "=== Labels ==="
    if [ "$show_values" = "true" ]; then
        extract_keys_values_by_frequency "labels" < "$tmpfile"
    else
        extract_keys_by_frequency "labels" < "$tmpfile"
    fi
fi

if [ "$show_annotations" = "true" ]; then
    if [ "$show_labels" = "true" ]; then
        echo ""
    fi
    echo "=== Annotations ==="
    if [ "$show_values" = "true" ]; then
        extract_keys_values_by_frequency "annotations" < "$tmpfile"
    else
        extract_keys_by_frequency "annotations" < "$tmpfile"
    fi
fi
