FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.20 AS builder
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG APP_NAME=unknown
ARG REVISION=unknown
ARG TAG=unknown
ARG BUILD_TIME=unknown

# Set the working directory inside the container
WORKDIR /build

# Steal the nobody user for the scratch image
RUN cat /etc/passwd | grep nobody > /etc/passwd_nobody

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./

# Copy the source code
COPY . .

RUN echo "Building for $TARGETOS/$TARGETARCH for $APP_NAME"

# -X github.com/cloudzero/cirrus-hex/app/internal/build.Rev=${REVISION}
# -X github.com/cloudzero/cirrus-hex/app/internal/build.Tag=${TAG}"
# -X github.com/cloudzero/cirrus-hex/app/internal/build.Time=${BUILD_TIME}
# Build the Go binary
RUN if [ "$TARGETARCH" = "amd64" ]; then \
        CCTARGET='zig cc -target x86_64-linux-musl' CXXTARGET='zig c++ -target x86_64-linux-musl'; \
    else \
        CCTARGET='zig cc -target aarch64-linux-musl' CXXTARGET='zig c++ -target aarch64-linux-musl'; \
    fi \
    && CGO_ENABLED=1 \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    CC=$CCTARGET \
    CXX=$CXXTARGET \
    go build \
    -mod=readonly \
    -trimpath \
    -ldflags="-s -w -X github.com/cloudzero/cirrus-hex/app/internal/build.Time=${BUILD_TIME} -X github.com/cloudzero/cirrus-hex/app/internal/build.Rev=${REVISION} -X github.com/cloudzero/cirrus-hex/app/internal/build.Tag=${APP_NAME}" \
    -tags 'netgo osusergo' \
    -o /build/$APP_NAME \
    /build/app/functions/$APP_NAME/main.go

# Stage 2: Access current certs
FROM gcr.io/distroless/static-debian11:debug@sha256:a0a404776dec98be120089ae42bbdfbe48c177921d856937d124d48eb8c0b951 AS certs

# TODO: use busybox for development - switch to "scratch" before release
#FROM busybox:1.36.1-uclibc
FROM scratch
ARG APP_NAME=unknown

# Ensure we have certs for HTTPS requests
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# TODO: Add default configuration file

# Copy the /etc/passwd file with the nobody user
COPY --from=builder /etc/passwd_nobody /etc/passwd

USER nobody

# Set the working directory inside the container
WORKDIR /cloudzero

LABEL org.opencontainers.image.title="cloudzero-${APP_NAME}"
LABEL org.opencontainers.image.description="cloudzero ${APP_NAME} application"
LABEL org.opencontainers.image.vendor="Cloudzero, Inc."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/Cloudzero/cloudzero-discovery"
LABEL io.artifacthub.package.readme-url="https://raw.githubusercontent.com/Cloudzero/cloudzero-discovery/main/README.md"
LABEL io.artifacthub.package.license="Apache-2.0"

EXPOSE 8080
VOLUME [ "/cloudzero/config" ]
ENV PATH=/app:$PATH

# Copy the Go binary from the builder stage
COPY --from=builder /build/$APP_NAME /cloudzero/app

# Run the Go binary
CMD ["/cloudzero/app"]