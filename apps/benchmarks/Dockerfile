FROM golang:1.25-alpine AS build-env

WORKDIR /src

COPY apps/benchmarks/go.mod apps/benchmarks/go.sum ./
RUN go mod download

COPY apps/benchmarks/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o ev-benchmarks .

FROM alpine:3.22.2

ENV TZ=UTC

#hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates curl tzdata

WORKDIR /root

COPY --from=build-env /src/ev-benchmarks /usr/bin/ev-benchmarks
COPY apps/benchmarks/matrices/baseline.json /root/baseline.json
COPY apps/benchmarks/matrices/burst.json /root/burst.json

CMD ["ev-benchmarks", "start"]
