FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Disable CGO for cross-platform compatibility and build
ENV CGO_ENABLED=0

# Download dependencies and build
RUN go mod download && go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]
