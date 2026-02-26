FROM golang:1.25 AS base

#hadolint ignore=DL3018
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# enable faster module downloading.
ENV GOPROXY=https://proxy.golang.org

## builder stage.
#
FROM base AS builder

WORKDIR /ev-node

# Copy all source code first
COPY . .

# Now download dependencies and build
RUN go mod download && cd apps/testapp && go install .

## prep the final image.
#
FROM base

COPY --from=builder /go/bin/testapp /usr/bin

WORKDIR /apps

ENTRYPOINT ["testapp"]
