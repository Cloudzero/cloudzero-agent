#!/usr/bin/env bash

set -euo pipefail

# Allow the namespace to be overridden by an environment variable
NAMESPACE="${NAMESPACE:-production}"

root_dir=$(git rev-parse --show-toplevel)

function create_namespace_if_not_exist() {
    if ! kubectl get namespace ${NAMESPACE} &> /dev/null; then
        echo "Creating ${NAMESPACE} namespace"
        kubectl create namespace ${NAMESPACE}
    else
        echo "Namespace ${NAMESPACE} already exists. No creation needed."
    fi
}

function create_certificates_if_not_exist() {
    if [[ ! -f "${root_dir}/certs/tls.key" || ! -f "${root_dir}/certs/tls.crt" ]]; then
        echo "Creating certificates"
        mkdir -p ${root_dir}/certs
        openssl genrsa -out ${root_dir}/certs/tls.key 2048
        openssl req -new -key ${root_dir}/certs/tls.key -out ${root_dir}/certs/tls.csr -subj "/CN=webhook-server.${NAMESPACE}.svc"
        openssl x509 -req -extfile <(printf "subjectAltName=DNS:webhook-server.${NAMESPACE}.svc") -in ${root_dir}/certs/tls.csr -signkey ${root_dir}/certs/tls.key -out ${root_dir}/certs/tls.crt
    else
        echo "Certificates already exist. No creation needed."
    fi
}

function install_certificate_secret_if_not_exist() {
    if ! kubectl get secret webhook-server-tls -n ${NAMESPACE} &> /dev/null; then
        echo "Creating Webhook Server TLS Secret in namespace ${NAMESPACE}"
        kubectl create secret tls webhook-server-tls \
            --cert "${root_dir}/certs/tls.crt" \
            --key "${root_dir}/certs/tls.key" -n ${NAMESPACE}
    else
        echo "TLS secret already exists in namespace ${NAMESPACE}. No creation needed."
    fi
}

create_certificates_if_not_exist
create_namespace_if_not_exist
install_certificate_secret_if_not_exist

echo "Creating Webhook Server Deployment in namespace ${NAMESPACE}"
kubectl create -f ${root_dir}/manifests/webhook_server.yml -n ${NAMESPACE}

echo "Creating K8s Webhooks in namespace ${NAMESPACE}"
ENCODED_CA=$(cat ${root_dir}/certs/tls.crt | base64 | tr -d '\n')
sed -e 's@${ENCODED_CA}@'"$ENCODED_CA"'@g' <"${root_dir}/manifests/webhooks.yml" | kubectl create -f -
