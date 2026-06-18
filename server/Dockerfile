FROM golang:1.26 AS builder
WORKDIR /usr/local/app

COPY go.mod ./ 
COPY go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -o server ./cmd/main.go

FROM alpine:latest
WORKDIR /usr/local/app

COPY --from=builder /usr/local/app/server ./

CMD ["./server"]
