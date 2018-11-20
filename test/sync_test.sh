#!/bin/bash

GDEX=../build/bin/gdex

logsdir=$PWD/sync-log
rm -rf $logsdir
mkdir $logsdir

# A standalone RPC server for accepting RPC requests.
for i in $(seq 0 3); do
  datadir=$PWD/Dexon.sync.$i
  rm -rf $datadir
  $GDEX --datadir=$datadir init genesis.json
  $GDEX --testnet --verbosity=4 --gcmode=archive --datadir=$datadir \
    --rpc --rpcapi=eth,net,web3,debug --rpcaddr=0.0.0.0 \
    --port=$((30505 + $i)) \
    --rpcport=$((8663 + $i *2)) \
    --wsport=$((8664 + $i * 2))  \
    --ws --wsapi=eth,net,web3,debug --wsaddr=0.0.0.0 \
    --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
    > $logsdir/gdex.sync.$i.log 2>&1 &
done

tail -f $logsdir/gdex.*.log
