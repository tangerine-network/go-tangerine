FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers g++ gmp-dev openssl-dev pkgconfig

ADD . /dexon
RUN cd /dexon && make clean && DOCKER=alpine make gdex all

# Pull Gdex into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates libstdc++ curl gmp openssl
COPY --from=builder /dexon/build/bin/gdex /usr/local/bin/
COPY --from=builder /dexon/build/bin/bootnode /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gdex"]
