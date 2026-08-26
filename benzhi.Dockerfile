FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies and build binary
RUN go mod download && CGO_ENABLED=0 go build -o /app/logalert-server ./cmd/server

# Start the server
CMD ["/app/logalert-server"]
