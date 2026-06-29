FROM golang:1.26-alpine AS builder

WORKDIR /app

# Enable network downloads instead of checking local paths
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ... lines 1 to 10 remain exactly the same ...

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 👇 ADD THIS LINE RIGHT HERE:
COPY migrations/ ./migrations/

RUN go build -o main ./cmd/api

EXPOSE 8080

CMD ["./main"]