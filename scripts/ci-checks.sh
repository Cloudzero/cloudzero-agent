#!/usr/bin/env bash

set -euo pipefail

FAILED=false

# Check Go version consistency
function check_go_version() {
    DESIRED_GO_VERSION="$(git grep -hPo '^go [0-9]+\.[0-9]+(\.[0-9]+)?$' go.mod | awk '{print $2}')"
    if [ -z "$DESIRED_GO_VERSION" ]; then
        echo "Error: No Go version found in go.mod" >&2
        exit 1
    fi
    DESIRED_GO_VERSION_NO_MICRO="$(echo $DESIRED_GO_VERSION | awk -F. '{print $1"."$2}')"

    # Dockerfiles
    for DOCKERFILE in \
            app/docker/Dockerfile \
            docker/Dockerfile \
            tests/docker/Dockerfile.collector \
            tests/docker/Dockerfile.shipper \
            tests/docker/Dockerfile.remotewrite \
            tests/integration/test_server/Dockerfile; do
        git grep -q " golang:${DESIRED_GO_VERSION_NO_MICRO}[- ]" ${DOCKERFILE} || {
            echo "${DOCKERFILE} does not have the desired Go version (${DESIRED_GO_VERSION_NO_MICRO})" >&2
            FAILED=true
        }
    done

    # go.mod
    for GO_MOD in go.mod app/tools/go.mod tests/integration/test_server/go.mod; do
        git grep -q "^go ${DESIRED_GO_VERSION}\$" ${GO_MOD} || {
            echo "${GO_MOD} does not have the desired Go version (${DESIRED_GO_VERSION})" >&2
            FAILED=true
        }
    done
}
check_go_version

if [ "$FAILED" = true ]; then
    exit 1
fi
