# Этап 1: Сборка приложения
FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/tester main.go

# Этап 2: Создание финального образа
FROM alpine:latest
COPY --from=builder /bin/tester /bin/tester
EXPOSE 8081
ENTRYPOINT ["/bin/tester"]