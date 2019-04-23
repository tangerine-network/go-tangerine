#!/bin/bash

accounts_opt="--account=0x`cat ../test/keystore/monkey.key`,100000000000000000000"

# node key's account
for key in ../test/keystore/test*key; do
    accounts_opt+=" --account=0x`cat $key`,1000000000000000000000"
done

git clone --depth 1 -b master https://github.com/dexon-foundation/governance-abi

# deploy contract
cd governance-abi
npm ci
./node_modules/.bin/ganache-cli -p 8645 -b 5 $accounts_opt > ../../test/ganache.log 2>&1 &
./node_modules/.bin/truffle migrate --network=smoke
