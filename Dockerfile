# ==========================================
# STAGE 1: Build the optimized Go binary executable
# ==========================================
FROM golang:1.26-alpine AS builder
# Install essential build tools inside the compiler sandbox
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency manifests first to leverage Docker layer caching speeds
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your application source code tree onto disk
COPY . .

# Compile the final binary with production optimizations:
# - CGO_ENABLED=0 creates a statically linked binary that runs anywhere
# - GOOS=linux targets the deployment system kernel architecture
# - ldflags="-w -s" strips out debug metadata and symbol tables to save space
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o vaultpay-api ./cmd/api/main.go

# ==========================================
# STAGE 2: Construct the lightweight scratch run container
# ==========================================
FROM alpine:3.19 AS runner
# Add security updates and root SSL CA certificates for safe HTTPS external network outbound calls
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copy only the compiled binary file over from our builder stage sandbox
COPY --from=builder /app/vaultpay-api .

# Copy your migrations folder into the runtime environment so migrations run on boot
COPY --from=builder /app/migrations ./migrations

# Expose our internet gateway port
EXPOSE 8080

# Execute our compiled engine
CMD ["./vaultpay-api"]