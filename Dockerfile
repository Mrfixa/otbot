# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled for sqlite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o otp-bot .

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/otp-bot .

# Copy config if exists
COPY config.yml.example /app/config.yml

# Create directories
RUN mkdir -p /app/data /app/logs && chown -R appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Run the bot
CMD ["./otp-bot", "-config", "/app/config.yml"]
