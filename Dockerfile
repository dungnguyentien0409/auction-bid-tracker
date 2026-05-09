# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache make

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /root/

# Copy binary and config
COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server"]
