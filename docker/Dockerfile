# export IMAGE_NAME=ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator
# export TAG=golang
# docker buildx build --builder=container --platform linux/amd64,linux/arm64 --push -t ${IMAGE_NAME}:${TAG} -f docker/Dockerfile .
#
# Stage 1: Build the Go binary
FROM golang:1.22 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# These values are replaced at compile time using the -X build flag:
#
# -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION}
# -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}"
# -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME}
ARG REVISION=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown
# LD_FLAGS="-s -w -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}"
# CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="${LD_FLAGS}" -tags 'netgo osusergo' -o cloudzero-agent-validator
# Build the Go binary
RUN CGO_ENABLED=0 go build \
    -mod=readonly \
    -trimpath \
    -ldflags="-s -w -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}" \
    -tags 'netgo osusergo' \
    -o /go/bin/cloudzero-agent-validator \
    cmd/cloudzero-agent-validator/main.go

# Stage 2: Access current certs
FROM gcr.io/distroless/static-debian11:debug@sha256:a0a404776dec98be120089ae42bbdfbe48c177921d856937d124d48eb8c0b951 AS certs

# Stage32: Create the distroless image
FROM busybox:1.36.1-uclibc

# Ensure we have certs for HTTPS requests
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER nobody

# Set the working directory inside the container
WORKDIR /app

LABEL org.opencontainers.image.title="cloudzero-agent-validator"
LABEL org.opencontainers.image.description="cloudzero-agent execution environment validation utility"
LABEL org.opencontainers.image.vendor="Cloudzero, Inc."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/Cloudzero/cloudzero-agent-validator"
LABEL io.artifacthub.package.readme-url="https://raw.githubusercontent.com/Cloudzero/cloudzero-agent-validator/main/README.md"
LABEL io.artifacthub.package.license="Apache-2.0"

VOLUME [ "/app/config" ]
ENV PATH=/app:$PATH

# Copy the Go binary from the builder stage
COPY --from=builder /go/bin/cloudzero-agent-validator /app/cloudzero-agent-validator

# Allow the default ENTRYPOINT from busybox to be the default,
# however run the app as the default command
CMD ["/app/cloudzero-agent-validator", "-h"]