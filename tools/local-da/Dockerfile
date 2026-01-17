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

WORKDIR /root

COPY --from=build-env /src/build/local-da /usr/bin/local-da

ENTRYPOINT ["/usr/bin/local-da"]
CMD ["-listen-all"]
