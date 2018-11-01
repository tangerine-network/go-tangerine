#!/bin/bash

GDEX=../build/bin/gdex

pkill -9 -f gdex

bootnode -nodekey bootnode.key --verbosity=9 > bootnode.log 2>&1 &

for i in $(seq 0 3); do
  datadir=$PWD/Dexon.$i
  rm -rf $datadir
  $GDEX --datadir=$datadir init genesis.json
  $GDEX --verbosity=4 --gcmode=archive --datadir=$datadir --nodekey=test$i.nodekey --port=$((28000 + $i)) --rpc --rpcaddr=0.0.0.0 --rpcport=$((8545 + $i * 2)) --rpcapi=eth,net,web3,debug --ws --wsapi=eth,net,web3,debug --wsaddr=0.0.0.0 --wsport=$((8546 + $i * 2)) --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" > gdex.$i.log 2>&1 &
done

tail -f gdex.*.log
