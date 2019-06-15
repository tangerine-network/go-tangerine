#!/bin/sh

fail()
{
  # name=ci-fail-$(date +%s).tar.gz
  # tar -zcvf $name test
  # echo "Verify fail and upload $name"
  # go run build/testtool/testtool.go upload $name dexon-prod-builds
  echo
}

endpoint=http://127.0.0.1:8545

timeout=300

echo "Wait for recovery"
if ! go run build/testtool/testtool.go waitForRecovery $endpoint $timeout; then
  fail
  exit 1
fi
