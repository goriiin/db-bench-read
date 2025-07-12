FROM golang:1.23.4-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY lib/ ./lib/
COPY db_etcd_test/seed/main.go ./main.go
COPY config.yaml ./
RUN go build -o seed main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/seed ./seed
COPY config.yaml ./
CMD ["./seed"]
