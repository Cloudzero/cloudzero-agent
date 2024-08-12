#!/usr/bin/env bash

set -euo pipefail

root_dir=$(git rev-parse --show-toplevel)

function create_namespace_if_not_exist() {
    if kubectl get namespace application &> /dev/null; then
        echo "Application namespace already exists. No creation needed."
    else
        echo "Creating application namespace"
        kubectl create namespace application
    fi
}

# Simplify command structure and correct the syntax
create_namespace_if_not_exist
kubectl apply -f ${root_dir}/manifests/test_deployment.yml -n application
