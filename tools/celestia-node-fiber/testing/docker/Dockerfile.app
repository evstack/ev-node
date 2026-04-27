# Build a celestia-appd binary with the `fibre` build tag enabled and a
# matching `fibre` server binary. Both go on PATH so the validator
# entrypoint can run them as separate processes.
#
# Pin CELESTIA_APP_REF to a feature/fibre commit; the default tracks
# whatever celestia-app `main` looks like at build time, which is where
# fibre development lives.
ARG GO_VERSION=1.26.1
ARG CELESTIA_APP_REPO=https://github.com/celestiaorg/celestia-app.git
ARG CELESTIA_APP_REF=main

FROM golang:${GO_VERSION}-bookworm AS build
ARG CELESTIA_APP_REPO
ARG CELESTIA_APP_REF
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone --depth 1 --branch "${CELESTIA_APP_REF}" "${CELESTIA_APP_REPO}" celestia-app \
    || git clone "${CELESTIA_APP_REPO}" celestia-app
WORKDIR /src/celestia-app
RUN git checkout "${CELESTIA_APP_REF}" || true
ENV CGO_ENABLED=0 GOFLAGS="-mod=readonly"
RUN go build -tags "ledger,fibre" -o /out/celestia-appd ./cmd/celestia-appd
RUN go build -tags "ledger,fibre" -o /out/fibre        ./fibre/cmd

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates jq curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/celestia-appd /usr/local/bin/celestia-appd
COPY --from=build /out/fibre         /usr/local/bin/fibre
RUN chmod +x /usr/local/bin/celestia-appd /usr/local/bin/fibre
WORKDIR /home/celestia
