# Step 1: Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -ldflags="-w -s" -o auth_service main.go

# Step 2: Final runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/auth_service /app/

# Create directory for persistent SQLite DB volume
RUN mkdir -p /app/data

# Environment defaults
ENV PORT=8084
ENV DB_PATH=/app/data/kather_baksho.db
ENV GIN_MODE=release

EXPOSE 8084

# Command to run microservice
CMD ["/app/auth_service"]
