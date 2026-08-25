FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build
ENV CGO_ENABLED=0
RUN go mod download && go build ./...

# Start the server
CMD ["go", "run", "./cmd/server"]