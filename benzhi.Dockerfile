FROM golang:1.22 AS builder

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build
RUN go mod download && CGO_ENABLED=0 go build -o /app/logalert-server ./cmd/server

# Runtime image
FROM golang:1.22

WORKDIR /app

# Copy the binary and web files
COPY --from=builder /app/logalert-server /app/logalert-server
COPY --from=builder /app/web /app/web

# Expose port
EXPOSE 8080

# Start the server
CMD ["/app/logalert-server"]
