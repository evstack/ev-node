FROM golang:1.26-bookworm AS build-env

WORKDIR /src

COPY core core

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    cd ./apps/evm && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/evm .


FROM busybox:1.36.1-musl AS busybox

FROM scratch


COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /out/evm /usr/bin/evm
COPY --from=build-env /src/apps/evm/entrypoint.sh /usr/bin/entrypoint.sh
COPY --from=busybox /bin/busybox /bin/busybox

ENV HOME=/home/ev-node
WORKDIR /home/ev-node
USER 10001:10001

ENTRYPOINT ["/bin/busybox", "sh", "/usr/bin/entrypoint.sh"]
