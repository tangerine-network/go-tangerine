#!/bin/bash

GDEX=../build/bin/gdex

pkill -9 -f gdex

bootnode -nodekey bootnode.key --verbosity=9 > bootnode.log 2>&1 &

for i in $(seq 0 3); do
  datadir=$PWD/Dexon.$i
  rm -rf $datadir
  $GDEX --datadir=$datadir init genesis.json
  $GDEX --verbosity=4 --datadir=$datadir --nodekey=test$i.nodekey --port=$((28000 + $i)) --rpc --rpcaddr 127.0.0.1 --rpcport=$((8545 + $i)) --rpcapi=eth,net,web3,debug --rpcvhosts='*' --rpccorsdomain="http://localhost:8000,https://www.myetherwallet.com" > gdex.$i.log 2>&1 &
done

tail -f gdex.*.log
