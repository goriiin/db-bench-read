FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/tester main.go

FROM alpine:latest
COPY --from=builder /bin/tester /bin/tester
EXPOSE 8081
ENTRYPOINT ["/bin/tester"]