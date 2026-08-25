FROM golang:1.22

WORKDIR /app

# Copy source code
COPY . /app

# Download dependencies
RUN go mod download

# Build the application (native build for current architecture)
RUN CGO_ENABLED=0 go build -o logalert-server ./cmd/server

# Start the server
CMD ["./logalert-server"]