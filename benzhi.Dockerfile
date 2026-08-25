# LogAlert Dockerfile
# Multi-stage build for production

# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod ./

# Copy source code
COPY . .

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /logalert ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /logalert /logalert

# Expose port
EXPOSE 8080

# Run the binary
CMD ["/logalert"]
