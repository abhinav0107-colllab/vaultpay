# --- Stage 1: Build Stage ---
FROM golang:1.26-alpine AS builder
# Install git and certificates (needed if fetching external dependencies)
RUN apk add --no-cache git ca-certificates

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker caching for dependencies
COPY go.mod ./
# If you have a go.sum file, uncomment the line below
# COPY go.sum ./
RUN go mod download

# Copy the entire project source code
COPY . .

# Build the Go application binary with optimizations for containers
# CGO_ENABLED=0 ensures static linking so it runs perfectly on a minimal alpine/scratch image
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o vaultpay ./cmd/api/main.go


# --- Stage 2: Final Runtime Stage ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/vaultpay .

# Expose the API port (adjust if your Go app runs on a different port)
EXPOSE 8080

# Run the binary
CMD ["./vaultpay"]