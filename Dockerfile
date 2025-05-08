FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o balancer ./cmd/main.go


FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/balancer .
COPY --from=builder /app/config ./config/
CMD ["./balancer", "-config", "./config/config.yaml"]