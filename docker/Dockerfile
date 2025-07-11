ARG DEPLOY_IMAGE=scratch

# Multi-stage Docker build with platform-specific cache optimization:
# 1. base-tools: Install system packages and tools (cached per platform)
# 2. dependencies: Download Go modules (cached per platform)
# 3. builder: Build Go binaries (cached per platform)
# 4. certs: Extract certificates from distroless image
# 5. final: Minimal runtime image with compiled binaries

# Stage 1: Base tools installation
FROM --platform=$BUILDPLATFORM golang:1.24.5-alpine AS base-tools
ARG TARGETPLATFORM
ARG TARGETOS TARGETARCH

# Set the working directory inside the container
WORKDIR /app

# Steal the nobody user for the scratch image
RUN grep nobody /etc/passwd > /etc/passwd_nobody

# Install system packages
RUN --mount=type=cache,target=/var/cache/apk,id=apk-$TARGETPLATFORM \
    apk add \
        binutils-gold \
        build-base \
        git \
        make \
        zig

# Stage 2: Dependencies stage
FROM base-tools AS dependencies
ARG TARGETPLATFORM
ARG TARGETOS TARGETARCH

# Copy Go module files first for better layer caching
COPY go.mod go.sum ./

# Download Go dependencies with platform-specific cache mount
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod-$TARGETPLATFORM \
    go mod download

# Stage 3: Build stage
FROM dependencies AS builder
ARG TARGETPLATFORM
ARG TARGETOS TARGETARCH
ARG REVISION=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown

# Copy only the files needed for building
#
# Note that this is very slimmed-down; hence the need to pass REGENERATE=never
# to the build command.
COPY app/ app/
COPY Makefile ./

# Build the Go binary with platform-specific cache mounts
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod-$TARGETPLATFORM \
    --mount=type=cache,target=/root/.cache/go-build,id=gobuild-$TARGETPLATFORM \
    --mount=type=cache,target=/root/.cache/zig,id=zig-$TARGETPLATFORM \
    make build OUTPUT_BIN_DIR=/go/bin \
    REGENERATE=never \
    TARGET_OS=$TARGETOS TARGET_ARCH=$TARGETARCH \
    ENABLE_ZIG=true \
    REVISION=${REVISION} \
    TAG=${TAG} \
    BUILD_TIME=${BUILD_TIME}

# Stage 4: Access current certs
FROM gcr.io/distroless/static-debian12:debug@sha256:112141358ce173d72800bb0e85b4d9dda7505ffc37203e70e475425d5a0d097b AS certs

# Stage 5: Final runtime image
# Note: For debugging, you can temporarily change the image used for building by
# passing in something like this to 'docker build':
#
#   --build-arg DEPLOY_IMAGE=busybox:latest-uclibc
#
# This is what the `package-debug` and `package-build-debug` Make targets do.
FROM ${DEPLOY_IMAGE}

# Ensure we have certs for HTTPS requests
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the /etc/passwd file with the nobody user
COPY --from=builder /etc/passwd_nobody /etc/passwd

USER nobody

# Set the working directory inside the container
WORKDIR /app

LABEL org.opencontainers.image.title="cloudzero-agent"
LABEL org.opencontainers.image.description="cloudzero-agent execution environment"
LABEL org.opencontainers.image.vendor="Cloudzero, Inc."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/Cloudzero/cloudzero-agent"
LABEL io.artifacthub.package.readme-url="https://raw.githubusercontent.com/Cloudzero/cloudzero-agent/main/README.md"
LABEL io.artifacthub.package.license="Apache-2.0"

VOLUME [ "/app/config" ]
ENV PATH=/app:$PATH

# Copy the Go binary from the builder stage
COPY --from=builder /go/bin/cloudzero-agent-inspector /app/cloudzero-agent-inspector
COPY --from=builder /go/bin/cloudzero-agent-validator /app/cloudzero-agent-validator
COPY --from=builder /go/bin/cloudzero-collector /app/cloudzero-collector
COPY --from=builder /go/bin/cloudzero-webhook /app/cloudzero-webhook
COPY --from=builder /go/bin/cloudzero-shipper /app/cloudzero-shipper
COPY --from=builder /go/bin/cloudzero-cluster-config /app/cloudzero-cluster-config
COPY --from=builder /go/bin/cloudzero-helmless /app/cloudzero-helmless
COPY --from=builder /go/bin/cloudzero-scout /app/cloudzero-scout

# Allow the default ENTRYPOINT from busybox to be the default,
# however run the app as the default command
CMD ["/app/cloudzero-agent-validator", "-h"]
