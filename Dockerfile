FROM golang:1.26-alpine AS builder

WORKDIR /app

# Enable network downloads instead of checking local paths
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/vaultpay-api ./cmd/api/main.go

FROM alpine:3.19 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/vaultpay-api .
EXPOSE 8080
CMD ["./vaultpay-api"]      