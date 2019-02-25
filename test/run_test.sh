#!/bin/bash

NETWORK="--bootnodes enode://0478aa13c91aa0db8e93b668313b7eb0532fbdb24f64772375373b14dbe326c238ad09ab4469f6442c9a9753f1275aeec2e531912c14a958ed1feb4ae7e227ef@127.0.0.1:30301"
GENESIS="genesis.json"

GDEX="../build/bin/gdex"
BOOTNODE="../build/bin/bootnode"

if [ ! -e "$BOOTNODE" ]; then
  echo "Building bootnode for the first time ..."
  go build -o $BOOTNODE ../cmd/bootnode
fi

# Start bootnode.
$BOOTNODE -nodekey keystore/bootnode.key --verbosity=9 > bootnode.log 2>&1 &

# Kill all previous instances.
pkill -9 -f gdex

logsdir=$PWD/log-$(date '+%Y-%m-%d-%H:%M:%S')
mkdir $logsdir

if [ -e log-latest ]; then
  rm -f log-previous
  mv log-latest log-previous
fi

rm -f log-latest
ln -s $logsdir log-latest

python << __FILE__
import re
import time

with open('$GENESIS', 'r') as f:
  data = f.read()

with open('$GENESIS', 'w') as f:
  dMoment = int(time.time()) + 15
  f.write(re.sub('"dMoment": [0-9]+,', '"dMoment": %d,' % dMoment, data))
__FILE__

# A standalone RPC server for accepting RPC requests.
datadir=$PWD/Dexon.rpc
rm -rf $datadir
$GDEX --datadir=$datadir init ${GENESIS}
$GDEX \
  ${NETWORK} \
  --verbosity=3 \
  --gcmode=archive \
  --datadir=$datadir --nodekey=keystore/rpc.key \
  --rpc --rpcapi=eth,net,web3,debug \
  --rpcaddr=0.0.0.0 --rpcport=8545 \
  --ws --wsapi=eth,net,web3,debug \
  --wsaddr=0.0.0.0 --wsport=8546  \
  --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
  > $logsdir/gdex.rpc.log 2>&1 &

# Nodes
for i in $(seq 0 3); do
  datadir=$PWD/Dexon.$i
  rm -rf $datadir
  $GDEX --datadir=$datadir init ${GENESIS}
  $GDEX \
    ${NETWORK} \
    --bp \
    --verbosity=4 \
    --gcmode=archive \
    --datadir=$datadir --nodekey=keystore/test$i.key \
    --port=$((30305 + $i)) \
    --rpc --rpcapi=eth,net,web3,debug \
    --rpcaddr=0.0.0.0 --rpcport=$((8547 + $i * 2)) \
    --ws --wsapi=eth,net,web3,debug \
    --wsaddr=0.0.0.0 --wsport=$((8548 + $i * 2)) \
    --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
    --pprof --pprofaddr=localhost --pprofport=$((6060 + $i)) \
    > $logsdir/gdex.$i.log 2>&1 &
done

if [ "$1" != "--ignore-log" ]; then
  tail -f $logsdir/gdex.*.log
fi
