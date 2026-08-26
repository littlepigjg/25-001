# Multi-stage Dockerfile for cross-platform (amd64/arm64) builds
# The builder stage runs on the native (host) architecture and cross-compiles
# for the target architecture, avoiding QEMU segfaults during Go compilation.

# Stage 1: Builder - runs on native arch, cross-compiles for target arch
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder

WORKDIR /app

# Copy source code
COPY . .

# Download dependencies and cross-compile for target platform
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 go mod download && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /server ./cmd/server

# Stage 2: Runtime - provides Go toolchain for verification + pre-built binary
FROM golang:1.22

WORKDIR /app

# Copy source code for go build/vet verification inside container
COPY . /app

# Also copy the pre-built binary
COPY --from=builder /server /app/server

EXPOSE 8080

# Default: run with go (will recompile from mounted source when -v is used)
CMD ["go", "run", "./cmd/server"]
