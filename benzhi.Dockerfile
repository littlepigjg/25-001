FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build
RUN go mod download && CGO_ENABLED=0 go build -o /usr/local/bin/server ./cmd/server

# Start the server
CMD ["/usr/local/bin/server"]
