FROM golang:1.22-alpine

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build
RUN go mod download && go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]
