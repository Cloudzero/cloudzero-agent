# export IMAGE_NAME=ghcr.io/cloudzero/cloudzero-insights-controller/cloudzero-insights-controller
# export TAG=golang
# docker buildx build --builder=container --platform linux/amd64,linux/arm64 --push -t ${IMAGE_NAME}:${TAG} -f docker/Dockerfile .
#
# Stage 1: Build the Go binary
# FROM golang:1.24 AS builder
FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.20 AS builder
ARG TARGETPLATFORM
ARG TARGETOS TARGETARCH
ARG REPO_NAME=cloudzero-insights-controller

# Set the working directory inside the container
WORKDIR /app

# Steal the nobody user for the scratch image
RUN cat /etc/passwd | grep nobody > /etc/passwd_nobody

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download \
    && apk add build-base git make zig

# Copy the source code
COPY . .

# These values are replaced at compile time using the -X build flag:
#
# -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Rev=${REVISION}
# -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Tag=${TAG}"
# -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Time=${BUILD_TIME}
ARG REVISION=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown
# LD_FLAGS="-s -w -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Tag=${TAG}"
# CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="${LD_FLAGS}" -tags 'netgo osusergo' -o cloudzero-insights-controller
# Build the Go binary
RUN make build OUTPUT_BIN_DIR=/go/bin \
    TARGETOS=$TARGETOS TARGETARCH=$TARGETARCH \
    ENABLE_ZIG=true \
    REPO_NAME=${REPO_NAME} \
    BUILD_TIME=${BUILD_TIME} \
    REVISION=${REVISION} TAG=${TAG}

# Stage 2: Access current certs
FROM gcr.io/distroless/static-debian11:debug@sha256:a0a404776dec98be120089ae42bbdfbe48c177921d856937d124d48eb8c0b951 AS certs

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

LABEL org.opencontainers.image.title="cloudzero-insights-controller"
LABEL org.opencontainers.image.description="cloudzero insights kubernetes validation admission controller"
LABEL org.opencontainers.image.vendor="Cloudzero, Inc."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/Cloudzero/cloudzero-insights-controller"
LABEL io.artifacthub.package.readme-url="https://raw.githubusercontent.com/Cloudzero/cloudzero-insights-controller/main/README.md"
LABEL io.artifacthub.package.license="Apache-2.0"

EXPOSE 8443
VOLUME [ "/app/config" ]
ENV PATH=/app:$PATH

# Copy the Go binary from the builder stage
COPY --from=builder /go/bin/cloudzero-insights-controller /app/cloudzero-insights-controller
COPY --from=builder /go/bin/cloudzero-insights-controller /app/controller
COPY --from=builder /go/bin/cloudzero-collector /app/cloudzero-collector
COPY --from=builder /go/bin/cloudzero-shipper /app/cloudzero-shipper

# Allow the default ENTRYPOINT from busybox to be the default,
# however run the app as the default command
CMD ["/app/cloudzero-insights-controller"]
