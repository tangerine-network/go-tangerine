#!/bin/bash -f

cd vendor/github.com/dexon-foundation/

sudo apt-get update

sudo apt-get -yq --no-install-suggests --no-install-recommends --allow-downgrades --allow-remove-essential --allow-change-held-packages install gcc-aarch64-linux-gnu libc6-dev-arm64-cross g++-aarch64-linux-gnu

rm -rf dep
mkdir dep; cd dep

echo  'travis_fold:start:aarch64-libgmp'
echo 'Cross compiling libgmp for aarch64'
mkdir libgmp; cd libgmp
wget https://gmplib.org/download/gmp/gmp-6.1.2.tar.bz2
tar -xjf gmp-6.1.2.tar.bz2
cd gmp-6.1.2
./configure --host=aarch64-linux-gnu --prefix=/usr/aarch64-linux-gnu --enable-cxx
sudo make -j8
sudo make install
cd ../..
echo  'travis_fold:end:aarch64-libgmp'

echo  'travis_fold:start:aarch64-openssl'
echo 'Cross compiling OpenSSL for aarch64'
mkdir openssl; cd openssl
wget https://www.openssl.org/source/openssl-1.1.1a.tar.gz
tar -xzf openssl-1.1.1a.tar.gz
cd openssl-1.1.1a
./Configure linux-aarch64 --prefix=/usr/aarch64-linux-gnu
sudo make CC=aarch64-linux-gnu-gcc -j8
sudo make install
cd ../..
echo  'travis_fold:end:aarch64-openssl'

cd ../

echo  'travis_fold:start:aarch64-bls'
echo 'Cross compiling bls for aarch64'
cd bls
make -C ../mcl clean
GMP_PREFIX=/usr/aarch64-linux-gnu/lib OPENSSL_PREFIX=/usr/aarch64-linux-gnu/lib ARCH=aarch64 CXX=aarch64-linux-gnu-g++ make clean all
echo  'travis_fold:end:aarch64-bls'
