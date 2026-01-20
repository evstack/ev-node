# Build stage
FROM golang:1.25-alpine AS builder

#hadolint ignore=DL3018
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /ev-node

# Copy go mod files
COPY go.mod go.sum ./
COPY apps/grpc/go.mod apps/grpc/go.sum ./apps/grpc/
COPY core/go.mod ./core/
COPY execution/grpc/go.mod execution/grpc/go.sum ./execution/grpc/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /ev-node/apps/grpc
RUN go build -o evgrpc .

# Runtime stage
FROM alpine:3.22.2

#hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 ev-node && \
    adduser -u 1000 -G ev-node -s /bin/sh -D ev-node

# Set working directory
WORKDIR /home/ev-node

# Copy binary from builder
COPY --from=builder /ev-node/apps/grpc/evgrpc /usr/local/bin/

# Create necessary directories
RUN mkdir -p /home/ev-node/.evgrpc && \
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
ENTRYPOINT ["evgrpc"]

# Default command
CMD ["start"]
