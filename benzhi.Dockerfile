FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build with CGO disabled for cross-platform compatibility
RUN CGO_ENABLED=0 go mod download && CGO_ENABLED=0 go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]
