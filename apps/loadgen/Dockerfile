FROM golang:1.25-alpine AS build-env

WORKDIR /src

COPY apps/loadgen/go.mod apps/loadgen/go.sum ./
RUN go mod download

COPY apps/loadgen/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o ev-loadgen .

FROM alpine:3.22.2

ENV TZ=UTC

#hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates curl tzdata

RUN addgroup -S ev && adduser -S ev -G ev

WORKDIR /home/ev

COPY --from=build-env /src/ev-loadgen /usr/bin/ev-loadgen
COPY apps/loadgen/matrices/baseline.json /home/ev/baseline.json
COPY apps/loadgen/matrices/burst.json /home/ev/burst.json

USER ev

CMD ["ev-loadgen", "start"]
