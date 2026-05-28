FROM golang:1.26-alpine AS build-env

WORKDIR /src

# TODO
# COPY go.mod go.sum ./
# RUN go mod download

# COPY . .

# WORKDIR /src/apps/evm
# RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o evm .

FROM alpine:3.22.2

ARG SUPERCRONIC_VERSION=0.2.46
ARG TARGETARCH

ENV TZ=UTC

#hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates curl tzdata

RUN wget -q "https://github.com/aptible/supercronic/releases/download/v${SUPERCRONIC_VERSION}/supercronic-linux-${TARGETARCH}" \
    -O /usr/local/bin/supercronic && \
    chmod +x /usr/local/bin/supercronic

WORKDIR /root

# TODO
# COPY --from=build-env /src/apps/benchmarks/ev-benchmarks /usr/bin/ev-benchmarks
COPY apps/benchmarks/entrypoint.sh /usr/bin/entrypoint.sh
COPY apps/benchmarks/crontab /etc/crontab
COPY apps/benchmarks/matrices/baseline.json /root/baseline.json

CMD ["supercronic", "/etc/crontab"]
