FROM golang:1.26.0-alpine AS builder

WORKDIR /build

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/bin/server ./internal/cmd/server

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /app/data /app/configs /app/certs

COPY --from=builder /build/bin/server ./server
COPY --from=builder /build/configs ./configs
COPY --from=builder /build/certs ./certs

VOLUME ["/app/data", "/app/configs", "/app/certs"]

EXPOSE 6001 7001

ENTRYPOINT ["./server", "-conf", "./configs"]
