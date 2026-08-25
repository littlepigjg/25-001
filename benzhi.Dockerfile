FROM golang:1.22

WORKDIR /app

ENV CGO_ENABLED=0

COPY . /app

RUN go mod download && go build ./...

CMD ["go", "run", "./cmd/server"]
