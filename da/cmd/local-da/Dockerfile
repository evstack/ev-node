FROM golang:1.24-alpine AS build-env

WORKDIR /src

RUN apk add --no-cache make

ADD core core

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build-da

FROM alpine:3.18.3

RUN apk --no-cache add ca-certificates curl

WORKDIR /root

COPY --from=build-env /src/build/local-da /usr/bin/local-da

ENTRYPOINT ["/usr/bin/local-da"]
CMD ["-listen-all"]
