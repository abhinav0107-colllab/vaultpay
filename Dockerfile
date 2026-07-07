# Step 1: Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Enable network downloads and fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Compile the binary with explicit absolute path outputs
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/vaultpay-api cmd/api/main.go

# Step 2: Final runtime stage
FROM alpine:3.19 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates

# Copy the binary explicitly from the absolute location
COPY --from=builder /app/vaultpay-api /app/vaultpay-api
COPY --from=builder /app/migrations /app/migrations

# Copy your crypto keys straight from the build workspace root
COPY --from=builder /app/private_key.pem /app/private_key.pem
COPY --from=builder /app/public_key.pem /app/public_key.pem

# Fix execution rights on alpine runner layers
RUN chmod +x /app/vaultpay-api

EXPOSE 8080
CMD ["/app/vaultpay-api"]