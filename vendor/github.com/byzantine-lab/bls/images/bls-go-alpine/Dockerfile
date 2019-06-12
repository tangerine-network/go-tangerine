FROM golang:alpine
MAINTAINER Jimmy Hu <jimmy.hu@dexon.org>

# Install dependencies
RUN apk add --update-cache build-base gmp-dev openssl-dev git

# Build bls library
RUN mkdir work ; cd work
RUN git clone --depth 1 git://github.com/dexon-foundation/mcl.git
RUN mkdir bls
COPY . bls/
RUN cd bls ; make clean && make test_go DOCKER=alpine -j && cp lib/* /usr/lib/
