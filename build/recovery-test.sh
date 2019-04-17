#!/bin/bash

tarAndUpload()
{
  name=travis-fail-$(date +%s).tar.gz
  tar -zcvf $name test
  echo "Verify fail and upload $name"
  PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go upload $name dexon-prod-builds
}

endpoint=http://127.0.0.1:8545

timeout=300

echo "Wait for recovery"
cmd="PKG_CONFIG_PATH=/usr/local/opt/openssl/lib/pkgconfig go run build/testtool/testtool.go waitForRecovery $endpoint $timeout"
eval $cmd
code=$?
if [ $code == 1 ]; then
  tarAndUpload
  exit 1
fi
