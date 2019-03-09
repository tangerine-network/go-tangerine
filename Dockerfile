# Build Gdex in a stock Go builder container
FROM golang:1.11-alpine3.9 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers g++ gmp-dev openssl-dev pkgconfig

ADD . /dexon
RUN cd /dexon && make clean && DOCKER=alpine make gdex

# Pull Gdex into a second stage deploy alpine container
FROM alpine:3.9

RUN apk add --no-cache ca-certificates libstdc++ curl gmp openssl
COPY --from=builder /dexon/build/bin/gdex /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gdex"]
