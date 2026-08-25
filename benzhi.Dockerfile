FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./

COPY . .

RUN go mod download && CGO_ENABLED=0 go build -o /logalert ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /logalert /app/logalert

EXPOSE 8080

CMD ["/app/logalert"]
