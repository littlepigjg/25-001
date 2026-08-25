FROM golang:1.22

WORKDIR /app

# Disable cgo for cross-platform compatibility (pure Go project)
ENV CGO_ENABLED=0

# Copy source code
COPY . /app

# Download dependencies and build
RUN go mod download && go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]
