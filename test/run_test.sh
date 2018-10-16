#!/bin/bash

GETH=../build/bin/geth

pkill -9 -f geth

bootnode -nodekey bootnode.key --verbosity=9 > bootnode.log 2>&1 &

for i in $(seq 0 3); do
  datadir=$PWD/Dexon.$i
  rm -rf $datadir
  $GETH --datadir=$datadir init genesis.json
  cp test$i.nodekey $datadir/geth/nodekey
  $GETH --verbosity=4 --datadir=$datadir --port=$((28000 + $i)) > geth.$i.log 2>&1 &
done

tail -f geth.*.log
