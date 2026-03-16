FROM golang:1.26-bookworm AS build-env

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/local-da ./tools/local-da

FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /out/local-da /usr/bin/local-da
USER 10001:10001

ENTRYPOINT ["/usr/bin/local-da"]
CMD ["-listen-all"]
