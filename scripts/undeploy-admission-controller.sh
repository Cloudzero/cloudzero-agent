#!/usr/bin/env bash

set -euo pipefail

# Allow the namespace to be overridden by an environment variable
NAMESPACE="${NAMESPACE:-cloudzero-insights}"

root_dir=$(git rev-parse --show-toplevel)

function delete_webhook_server_deployment() {
    echo "Deleting Webhook Server Deployment in namespace ${NAMESPACE}"
    kubectl delete -f ${root_dir}/manifests/webhook_server.yml -n ${NAMESPACE} --ignore-not-found
}

function delete_certificate_secret() {
    echo "Deleting Webhook Server TLS Secret in namespace ${NAMESPACE}"
    kubectl delete secret webhook-server-tls -n ${NAMESPACE} --ignore-not-found
}

function delete_k8s_webhooks() {
    echo "Deleting K8s Webhooks in namespace ${NAMESPACE}"
    sed -e 's@${ENCODED_CA}@'"$(cat ${root_dir}/certs/tls.crt | base64 | tr -d '\n')"'@g' <"${root_dir}/manifests/webhooks.yml" | kubectl delete -f - --ignore-not-found
}

function optional_delete_certificates() {
    read -p "Do you want to delete the local certificates? (y/n) " answer
    case "${answer}" in
        [Yy]* )
            echo "Deleting certificates"
            rm -rf ${root_dir}/certs
            ;;
        * )
            echo "Keeping certificates"
            ;;
    esac
}

function delete_namespace() {
    read -p "Are you sure you want to delete the namespace '${NAMESPACE}'? This will remove all resources within the namespace. (y/n) " confirm
    if [[ "${confirm}" =~ ^[Yy]$ ]]; then
        echo "Deleting namespace ${NAMESPACE}"
        kubectl delete namespace ${NAMESPACE} --ignore-not-found
    else
        echo "Namespace deletion aborted."
    fi
}

delete_webhook_server_deployment
delete_certificate_secret
delete_k8s_webhooks
optional_delete_certificates
delete_namespace
