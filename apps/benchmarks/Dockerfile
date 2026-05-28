FROM golang:1.25-alpine AS build-env

WORKDIR /src

COPY apps/benchmarks/go.mod apps/benchmarks/go.sum ./
RUN go mod download

COPY apps/benchmarks/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o ev-benchmarks .

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

COPY --from=build-env /src/ev-benchmarks /usr/bin/ev-benchmarks
COPY apps/benchmarks/crontab /etc/crontab
COPY apps/benchmarks/matrices/baseline.json /root/baseline.json
COPY apps/benchmarks/matrices/burst.json /root/burst.json

CMD ["supercronic", "/etc/crontab"]
