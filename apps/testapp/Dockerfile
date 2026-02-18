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
RUN go mod download && make install

## prep the final image.
#
FROM base

RUN groupadd -g 1000 ev-node && \
    useradd -u 1000 -g ev-node -s /bin/sh -m ev-node

COPY --from=builder /go/bin/testapp /usr/bin

WORKDIR /home/ev-node
RUN chown -R ev-node:ev-node /home/ev-node

USER ev-node

ENTRYPOINT ["testapp"]
