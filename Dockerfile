# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache make

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build-time arguments (validation only; runtime values injected via docker-compose)
ARG APP_ENV=production
ARG REPO_TYPE=redis

# Build the application
RUN APP_ENV=${APP_ENV} REPO_TYPE=${REPO_TYPE} make build

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /root/

# Copy binary and config
COPY --from=builder /app/bin/auction-bid-tracker ./server
COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server"]
