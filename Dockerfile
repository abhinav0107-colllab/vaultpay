FROM golang:1.26-alpine AS builder

WORKDIR /app

# Enable network downloads instead of checking local paths
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ... lines 1 to 10 remain exactly the same ...

# ... lines 1 to 12 stay exactly the same ...

FROM alpine:3.19 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/vaultpay-api .

# 👇 ADD THIS EXACT LINE RIGHT HERE:
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./vaultpay-api"]