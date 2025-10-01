# Build stage
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN go build -o worker-bin ./cmd/worker
RUN go build -o router ./cmd/router

# Runtime stage
FROM alpine:3.22

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh agent

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/worker-bin /app/worker-bin
COPY --from=builder /app/router /app/router

# Create hypotheses and artifacts directories
RUN mkdir -p /app/hypotheses /app/artifacts && chown agent:agent /app/hypotheses /app/artifacts

# Switch to non-root user
USER agent

# Expose ports
EXPOSE 8080 8081 8082

# Default command (can be overridden)
CMD ["./worker-bin"]
