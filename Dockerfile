# Step 1: Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Enable network downloads and fetch dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Compile the binary cleanly
RUN CGO_ENABLED=0 GOOS=linux go build -o vaultpay-api ./cmd/api/main.go

# Step 2: Final runtime stage
FROM alpine:3.19 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates

# Copy the binary and migrations directory from the builder workspace
COPY --from=builder /app/vaultpay-api .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
CMD ["./vaultpay-api"]