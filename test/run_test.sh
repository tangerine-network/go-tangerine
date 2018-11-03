#!/bin/bash

GDEX=../build/bin/gdex

# Kill all previous instances.
pkill -9 -f gdex

# Start bootnode.
bootnode -nodekey bootnode.key --verbosity=9 > bootnode.log 2>&1 &

logsdir=$PWD/log-$(date '+%Y-%m-%d-%H:%M:%S')
mkdir $logsdir

rm -f log-latest
ln -s $logsdir log-latest

# A standalone RPC server for accepting RPC requests.
datadir=$PWD/Dexon.rpc
rm -rf $datadir
$GDEX --datadir=$datadir init genesis.json
$GDEX --verbosity=4 --gcmode=archive --datadir=$datadir \
  --rpc --rpcapi=eth,net,web3,debug --rpcaddr=0.0.0.0 --rpcport=8543 \
  --ws --wsapi=eth,net,web3,debug --wsaddr=0.0.0.0 --wsport=8544  \
  --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
  > $logsdir/gdex.rpc.log 2>&1 &

# Nodes
for i in $(seq 0 3); do
  datadir=$PWD/Dexon.$i
  rm -rf $datadir
  $GDEX --datadir=$datadir init genesis.json
  $GDEX --verbosity=4 --gcmode=archive \
    --bp \
    --datadir=$datadir --nodekey=test$i.nodekey \
    --port=$((30305 + $i)) \
    --rpc --rpcapi=eth,net,web3,debug \
    --rpcaddr=0.0.0.0 --rpcport=$((8545 + $i * 2)) \
    --ws --wsapi=eth,net,web3,debug \
    --wsaddr=0.0.0.0 --wsport=$((8546 + $i * 2)) \
    --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
    > $logsdir/gdex.$i.log 2>&1 &
done

tail -f $logsdir/gdex.*.log
