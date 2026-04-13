# Build context must be the repo root (not operator/) because the go.mod replace
# directive points to ../ and the builder needs access to the root module source.
#
# Example build command (from repo root):
#   docker build -f operator/Dockerfile -t cloudzero-agent-operator:latest .
FROM golang:1.26.1-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy root module first for better layer caching.
COPY go.mod go.sum ./
COPY app/ app/

# Copy operator module.
COPY operator/go.mod operator/go.sum operator/
COPY operator/cmd/ operator/cmd/
COPY operator/api/ operator/api/
COPY operator/internal/ operator/internal/
COPY operator/hack/ operator/hack/

WORKDIR /workspace/operator
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -o manager ./cmd/

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/operator/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
