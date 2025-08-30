# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /ev-node

# Copy go mod files
COPY go.mod go.sum ./
COPY apps/grpc/single/go.mod apps/grpc/single/go.sum ./apps/grpc/single/
COPY core/go.mod core/go.sum ./core/
COPY execution/grpc/go.mod execution/grpc/go.sum ./execution/grpc/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY sequencers/single/go.mod sequencers/single/go.sum ./sequencers/single/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /ev-node/apps/grpc/single
RUN go build -o grpc-single .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 ev-node && \
    adduser -u 1000 -G ev-node -s /bin/sh -D ev-node

# Set working directory
WORKDIR /home/ev-node

# Copy binary from builder
COPY --from=builder /ev-node/apps/grpc/single/grpc-single /usr/local/bin/

# Create necessary directories
RUN mkdir -p /home/ev-node/.grpc-single && \
    chown -R ev-node:ev-node /home/ev-node

# Switch to non-root user
USER ev-node

# Expose ports
# P2P port
EXPOSE 26656
# RPC port
EXPOSE 26657
# Prometheus metrics
EXPOSE 26660

# Set entrypoint
ENTRYPOINT ["grpc-single"]

# Default command
CMD ["start"]
