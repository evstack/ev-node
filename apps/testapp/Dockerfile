FROM golang:1.26-bookworm AS builder

WORKDIR /ev-node

COPY go.mod go.sum ./
COPY apps/testapp/go.mod apps/testapp/go.sum ./apps/testapp/
COPY core/go.mod core/go.sum ./core/
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN mkdir -p /home/ev-node
RUN --mount=type=cache,target=/go/pkg/mod \
   cd ./apps/testapp &&  \
   CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/testapp .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=10001:10001 /home/ev-node /home/ev-node
COPY --from=builder /out/testapp /usr/bin/testapp

WORKDIR /home/ev-node

USER 10001:10001

ENTRYPOINT ["testapp"]
