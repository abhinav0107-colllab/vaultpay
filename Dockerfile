# ==========================================
# STAGE 1: Build the optimized Go binary executable
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy your vendor directory onto disk cleanly
COPY go.mod go.sum ./
COPY vendor/ ./vendor/

# Copy the rest of your application source code tree onto disk
COPY . .

# Compile the final binary with production vendor optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/vaultpay-api ./cmd/api/main.go

# ==========================================
# STAGE 2: Construct the lightweight scratch run container
# ==========================================
FROM alpine:3.19 AS runner

WORKDIR /app

# Add security updates and root SSL CA certificates for safe HTTPS outbound calls
RUN apk add --no-cache ca-certificates

# Pull the compiled binary file from the builder sandbox stage layer
COPY --from=builder /app/vaultpay-api .

# Expose your application network listener port mapping
EXPOSE 8080

# Execute the application binary on container startup
CMD ["./vaultpay-api"]