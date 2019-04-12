FROM golang:1.12-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers g++ gmp-dev pkgconfig

ADD . /dexon
RUN cd /dexon && make clean-cgo && DOCKER=alpine make gdex
RUN cd /dexon && build/env.sh go build -o build/bin/bootnode ./cmd/bootnode

# Pull Gdex into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates curl libstdc++ gmp
COPY --from=builder /dexon/build/bin/gdex /usr/local/bin/
COPY --from=builder /dexon/build/bin/bootnode /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["gdex"]
