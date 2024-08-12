#!/usr/bin/env bash

set -euo pipefail

root_dir=$(git rev-parse --show-toplevel)

function delete_test_deployment() {
    echo "Deleting test deployment from application namespace"
    kubectl delete -f ${root_dir}/manifests/test_deployment.yml -n application --ignore-not-found
}

function delete_namespace() {
    read -p "Are you sure you want to delete the 'application' namespace? All resources within will be removed. (y/n) " confirm
    if [[ "${confirm}" =~ ^[Yy]$ ]]; then
        echo "Deleting 'application' namespace"
        kubectl delete namespace application --ignore-not-found
    else
        echo "Namespace deletion aborted."
    fi
}

delete_test_deployment
delete_namespace
