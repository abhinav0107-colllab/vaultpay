
FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY vendor/ ./vendor/

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/vaultpay-api ./cmd/api/main.go

FROM alpine:3.20 AS runner
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/vaultpay-api .

COPY --from=builder /app/migrations ./migrations

COPY --from=builder /app/private_key.pem .
COPY --from=builder /app/public_key.pem .

# Expose our internet gateway port
EXPOSE 8080

# Execute our compiled engine
CMD ["./vaultpay-api"]