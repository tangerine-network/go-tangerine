# Build Geth in a stock Go builder container
FROM golang:1.11-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers g++ gmp-dev openssl-dev

ADD . /dexon
RUN cd /dexon && DOCKER=alpine make gdex

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates libstdc++
COPY --from=builder /dexon/build/bin/gdex /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gdex"]
