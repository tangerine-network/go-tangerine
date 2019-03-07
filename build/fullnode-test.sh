#!/bin/bash

sleep 10

tarAndUpload()
{
  name=travis-fail-$(date +%s).tar.gz
  tar -zcvf $name test
  echo "Verify fail and upload $name"
  PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go upload $name dexon-builds
}

endpoint=http://127.0.0.1:8545

for round in 0 1 2 3 4
do

echo "Start verify round $round"
    for index in 0 1 2 3
    do
    echo "Verify gov master public key round $round index $index"
    cmd="PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go verifyGovMPK $endpoint $round $index"
    eval $cmd
    code=$?

    if [ $code == 1 ]; then
      tarAndUpload
      exit 1
    fi
    done

echo "Start verify CRS"
cmd="PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go verifyGovCRS $endpoint $round"
eval $cmd
code=$?

if [ $code == 1 ]; then
  tarAndUpload
  exit 1
fi

if [ $round -lt 4 ]; then
  cmd="PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go monkeyTest $endpoint"
  eval $cmd
  code=$?

  if [ $code == 1 ]; then
    tarAndUpload
    exit 1
  fi

  echo "Sleep 15 sec wait for next round"
  sleep 15
fi
done
