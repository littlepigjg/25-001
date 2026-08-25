FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build (CGO_ENABLED=0 for cross-platform compatibility)
RUN go mod download && CGO_ENABLED=0 go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]
