#!/bin/bash

rm -rf output
mkdir output
pushd ./output

git clone git@github.com:leptonai/xray-core.git
pushd ./xray-core/main
GOOS=linux GOARCH=amd64 go build -o ../../xray -v
popd

git clone git@github.com:leptonai/lepton.git
pushd ./lepton/xray-manager
GOOS=linux GOARCH=amd64 go build -o ../../xray-manager -v
popd

mkdir xray-linux-amd64
outputDir="./xray-linux-amd64/"
cp -r ../conf.d ${outputDir}
cp ../*.dat ${outputDir}
cp ../{*.bash,*.service} ${outputDir}
cp ./{xray,xray-manager} ${outputDir}

TAR=${TAR:-`which tar`}
${TAR} czvf xray-linux-amd64.tar.gz ./xray-linux-amd64

popd

