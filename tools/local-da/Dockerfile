FROM golang:1.25-alpine AS build-env

WORKDIR /src

#hadolint ignore=DL3018
RUN apk add --no-cache make

COPY core core

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build-da

FROM alpine:3.22.2

#hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates curl

RUN addgroup -g 1000 ev-node && \
    adduser -u 1000 -G ev-node -s /bin/sh -D ev-node

WORKDIR /home/ev-node
RUN chown -R ev-node:ev-node /home/ev-node

COPY --from=build-env /src/build/local-da /usr/bin/local-da

USER ev-node

ENTRYPOINT ["/usr/bin/local-da"]
CMD ["-listen-all"]
