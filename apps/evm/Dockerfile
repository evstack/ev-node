FROM golang:1.25-alpine AS build-env

WORKDIR /src

COPY core core

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /src/apps/evm
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o evm .

FROM alpine:3.22.2

#hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates curl

RUN addgroup -g 1000 ev-node && \
    adduser -u 1000 -G ev-node -s /bin/sh -D ev-node

WORKDIR /home/ev-node

COPY --from=build-env /src/apps/evm/evm /usr/bin/evm
COPY apps/evm/entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh && \
    chown -R ev-node:ev-node /home/ev-node

USER ev-node

ENTRYPOINT ["/usr/bin/entrypoint.sh"]
