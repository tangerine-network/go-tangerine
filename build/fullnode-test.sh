#!/bin/sh

sleep 10

fail()
{
  # name=ci-fail-$(date +%s).tar.gz
  # tar -zcvf $name test
  # echo "Verify fail and upload $name"
  # go run build/testtool/testtool.go upload $name dexon-prod-builds
  echo
}

endpoint=http://127.0.0.1:8545

for round in 0 1 2 3 4
do

echo "Start verify round $round"
    for index in 0 1 2 3
    do
    echo "Verify gov master public key round $round index $index"
    if ! go run build/testtool/testtool.go verifyGovMPK $endpoint $round $index; then
      fail
      exit 1
    fi
    done

echo "Start verify CRS"
if ! go run build/testtool/testtool.go verifyGovCRS $endpoint $round; then
  fail
  exit 1
fi

if [ $round -lt 4 ]; then
  if ! go run build/testtool/testtool.go monkeyTest $endpoint; then
    fail
    exit 1
  fi

  echo "Sleep 15 sec wait for next round"
  sleep 15
fi
done
