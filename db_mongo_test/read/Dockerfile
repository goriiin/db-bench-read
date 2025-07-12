FROM golang:1.21-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY lib/ ./lib/
COPY db_mongo_test/read/main.go ./main.go
COPY config.yaml ./
RUN go build -o read main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/read ./read
COPY config.yaml ./
CMD ["./read"]
