# Step 1: Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Pure-Go SQLite driver (github.com/glebarez/sqlite) — no C toolchain needed.
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -ldflags="-w -s" -o order_service main.go

# Step 2: Final runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Create directory for persistent SQLite DB volume
RUN mkdir -p /app/data

# Copy compiled binary from builder stage
COPY --from=builder /app/order_service /app/

# Environment defaults
ENV PORT=8081
ENV DB_PATH=/app/data/orders.db
ENV GIN_MODE=release

EXPOSE 8081

# Command to run order-service
CMD ["/app/order_service"]
