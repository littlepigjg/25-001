FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build (CGO disabled for cross-platform compatibility)
ENV CGO_ENABLED=0
RUN go mod download && go build ./...

# Start the server (CGO already disabled via ENV above)
CMD ["go", "run", "./cmd/server"]
