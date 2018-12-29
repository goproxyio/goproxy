#!/usr/bin/env bash

PKG=${PWD}/internal
GOROOT=`go env GOROOT`

mkdir -p ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/* ${PKG}


cp -r ${GOROOT}/src/cmd/internal/browser ${PKG}
cp -r ${GOROOT}/src/cmd/internal/buildid ${PKG}
cp -r ${GOROOT}/src/cmd/internal/objabi ${PKG}
cp -r ${GOROOT}/src/cmd/internal/test2json ${PKG}

cp -r ${GOROOT}/src/internal/singleflight ${PKG}
cp -r ${GOROOT}/src/internal/testenv ${PKG}


find ${PKG} -type f -name '*.go' -exec sed -i 's/cmd\/go\/internal/github.com\/goproxyio\/goproxy\/internal/g' {} +
find ${PKG} -type f -name '*.go' -exec sed -i 's/cmd\/internal/github.com\/goproxyio\/goproxy\/internal/g' {} +
find ${PKG} -type f -name '*.go' -exec sed -i 's/internal\/singleflight/github.com\/goproxyio\/goproxy\/internal\/singleflight/g' {} +
find ${PKG} -type f -name '*.go' -exec sed -i 's/internal\/testenv/github.com\/goproxyio\/goproxy\/internal\/testenv/g' {} +