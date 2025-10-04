# Build stage
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o llmrouter \
    ./cmd/llmrouter

# Final stage - distroless
FROM gcr.io/distroless/static-debian11

# Copy the binary from builder stage
COPY --from=builder /app/llmrouter /llmrouter

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set working directory
WORKDIR /

# Expose port
EXPOSE 8090

# Set environment variables
ENV LLMROUTER_PORT=8090
ENV CONFIG=router.yaml

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/llmrouter", "healthcheck"]

# Run the binary
ENTRYPOINT ["/llmrouter"]