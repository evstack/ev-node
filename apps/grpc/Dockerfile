FROM golang:1.26-bookworm AS builder

WORKDIR /ev-node

COPY go.mod go.sum ./
COPY apps/grpc/go.mod apps/grpc/go.sum ./apps/grpc/
COPY core/go.mod core/go.sum ./core/
COPY execution/grpc/go.mod execution/grpc/go.sum ./execution/grpc/

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN mkdir -p /home/ev-node
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/evgrpc ./apps/grpc

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chown=10001:10001 /home/ev-node /home/ev-node
WORKDIR /home/ev-node

COPY --from=builder /out/evgrpc /usr/local/bin/evgrpc
USER 10001:10001

EXPOSE 26656
EXPOSE 26657
EXPOSE 26660

ENTRYPOINT ["evgrpc"]
CMD ["start"]
