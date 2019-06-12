#!/bin/bash

BOOTNODE_FLAGS="--bootnodes enode://b0dacdaceb9ce26f89406e8048d279d3aa81c770e967db7e2556e416ca446de0e9327dbdf85eb56c421eeabbc843ceb8f373e7a26dc31d48178620e48cb095c4@127.0.0.1:30301"
GENESIS="genesis.json"

GDEX="../build/bin/gtan"
BOOTNODE="../build/bin/bootnode"


CONTINUE=false
SMOKETEST=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --continue)
    CONTINUE=true
    ;;
    --smoke-test)
    SMOKETEST=true
    ;;
  esac
  shift
done


if [ ! -e "$BOOTNODE" ]; then
  echo "Building bootnode for the first time ..."
  go build -o $BOOTNODE ../cmd/bootnode
fi

# Start bootnode.
$BOOTNODE -nodekey keystore/bootnode.key --verbosity=9 > bootnode.log 2>&1 &

# Kill all previous instances.
pkill -9 -f gtan

logsdir=$PWD/log-$(date '+%Y-%m-%d-%H:%M:%S')
mkdir $logsdir

if [ -e log-latest ]; then
  rm -f log-previous
  mv log-latest log-previous
fi

rm -f log-latest
ln -s $logsdir log-latest


# the recovery contract address 0x80859F3d0D781c2c4126962cab0c977b37820e78 is deployed using keystore/monkey.key
if $SMOKETEST; then
  if [ `uname` == "Darwin" ]; then
    sed -i '' 's/"contract":.*,/"contract": "0x80859F3d0D781c2c4126962cab0c977b37820e78",/g' genesis.json
  else
    sed -i 's/"contract":.*,/"contract": "0x80859F3d0D781c2c4126962cab0c977b37820e78",/g' genesis.json
  fi
fi


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
datadir=$PWD/Tangerine.rpc
if ! $CONTINUE; then
  rm -rf $datadir
  $GDEX --datadir=$datadir init ${GENESIS}
fi
$GDEX \
  ${BOOTNODE_FLAGS} \
  --verbosity=3 \
  --nat=none \
  --gcmode=archive \
  --datadir=$datadir --nodekey=keystore/rpc.key \
  --rpc --rpcapi=eth,net,web3,debug \
  --rpcaddr=0.0.0.0 --rpcport=8545 \
  --ws --wsapi=eth,net,web3,debug \
  --wsaddr=0.0.0.0 --wsport=8546  \
  --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
  > $logsdir/gtan.rpc.log 2>&1 &

NUM_NODES=$(cat ${GENESIS} | grep 'Tangerine Test Node' | wc -l)

RECOVERY_FLAGS="--recovery.network-rpc=https://rinkeby.infura.io"

if $SMOKETEST; then
  RECOVERY_FLAGS="--recovery.network-rpc=http://127.0.0.1:8645"
fi


# Nodes
for i in $(seq 0 $(($NUM_NODES - 1))); do
  datadir=$PWD/Tangerine.$i

  if ! $CONTINUE; then
    rm -rf $datadir
    $GDEX --datadir=$datadir init ${GENESIS}
  fi
  $GDEX \
    ${BOOTNODE_FLAGS} \
    --bp \
    --verbosity=4 \
    --nat=none \
    --gcmode=archive \
    --datadir=$datadir --nodekey=keystore/test$i.key \
    --port=$((30305 + $i)) \
    ${RECOVERY_FLAGS} \
    --rpc --rpcapi=eth,net,web3,debug \
    --rpcaddr=0.0.0.0 --rpcport=$((8547 + $i * 2)) \
    --ws --wsapi=eth,net,web3,debug \
    --wsaddr=0.0.0.0 --wsport=$((8548 + $i * 2)) \
    --wsorigins='*' --rpcvhosts='*' --rpccorsdomain="*" \
    --pprof --pprofaddr=localhost --pprofport=$((6060 + $i)) \
    > $logsdir/gtan.$i.log 2>&1 &
done

if ! $SMOKETEST; then
  tail -f $logsdir/gtan.*.log
fi
