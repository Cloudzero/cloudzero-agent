// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package build

// These values are replaced at compile time using the -X build flag:
//
//	-X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION}
//	-X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}"
//	-X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME}
//
// Example:
//   BUILD_TIME="$(date -u '+%Y-%m-%d_%I:%M:%S%p')"
//   TAG="current"
//   REVISION="current"
//   if hash git 2>/dev/null && [ -e ${BDIR}/.git ]; then
//     TAG="$(git describe --tags 2>/dev/null || true)"
//     [[ -z "$TAG" ]] && TAG="notag"
//     REVISION="$(git rev-parse HEAD)"
//   fi
//
//   LD_FLAGS="-s -w -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}"
//   CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="${LD_FLAGS}" -tags 'netgo osusergo' -o cloudzero-agent-validator

var (
	Rev  = "latest"
	Tag  = "latest"
	Time = "latest"
)
