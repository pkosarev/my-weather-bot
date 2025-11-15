FROM golang:1.25.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/my-weather-bot .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/my-weather-bot .

CMD ["./my-weather-bot"]