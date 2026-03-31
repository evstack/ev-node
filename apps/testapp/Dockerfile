FROM golang:1.26 AS base

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

# Copy module files first to leverage Docker layer caching.
# Dependencies are only re-downloaded when go.mod or go.sum change.
COPY go.mod go.sum ./
COPY apps/testapp/go.mod apps/testapp/go.sum ./apps/testapp/
RUN go mod download && (cd apps/testapp && go mod download)

# Copy the rest of the source and build.
COPY . .

WORKDIR /ev-node/apps/testapp
RUN go build -o /go/bin/testapp .

## prep the final image.
#
FROM base

COPY --from=builder /go/bin/testapp /usr/bin

WORKDIR /apps

ENTRYPOINT ["testapp"]
