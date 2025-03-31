# Stage 1: Build the Go binary
FROM --platform=$BUILDPLATFORM golang:1.24.1-alpine AS builder
ARG TARGETPLATFORM
ARG TARGETOS TARGETARCH
ARG REVISION=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown

# Set the working directory inside the container
WORKDIR /app

# Steal the nobody user for the scratch image
RUN grep nobody /etc/passwd > /etc/passwd_nobody

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download \
    && apk add build-base make zig

# Copy the source code
COPY . .

# Build the Go binary
RUN make build OUTPUT_BIN_DIR=/go/bin \
    TARGET_OS=$TARGETOS TARGET_ARCH=$TARGETARCH \
    ENABLE_ZIG=true \
    REVISION=${REVISION} \
    TAG=${TAG} \
    BUILD_TIME=${BUILD_TIME}

# Stage 2: Access current certs
FROM gcr.io/distroless/static-debian12:debug@sha256:5c474275684bbf271ec40502ab50158b2f9826de5877d8feec27e22e8d6ee3d2 AS certs

# TODO: use busybox for development - switch to "scratch" before release
#FROM busybox:1.36.1-uclibc
FROM scratch

# Ensure we have certs for HTTPS requests
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# TODO: Add default configuration file

# Copy the /etc/passwd file with the nobody user
COPY --from=builder /etc/passwd_nobody /etc/passwd

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
COPY --from=builder /go/bin/cloudzero-agent-inspector /app/cloudzero-agent-inspector
COPY --from=builder /go/bin/cloudzero-agent-validator /app/cloudzero-agent-validator

# Allow the default ENTRYPOINT from busybox to be the default,
# however run the app as the default command
CMD ["/app/cloudzero-agent-validator", "-h"]
