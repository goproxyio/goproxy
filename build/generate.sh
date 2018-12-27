#!/usr/bin/env bash

PKG=${PWD}/pkg/
GOROOT=`go env GOROOT`
cp -r ${GOROOT}/src/cmd/go/internal/base ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/cache ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/cfg ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/dirhash ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/get ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/load ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/modfetch ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/modfile ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/modinfo ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/module ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/search ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/par ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/semver ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/txtar ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/str ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/web ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/web2 ${PKG}
cp -r ${GOROOT}/src/cmd/go/internal/work ${PKG}

cp -r ${GOROOT}/src/cmd/internal/browser ${PKG}
cp -r ${GOROOT}/src/cmd/internal/buildid ${PKG}
cp -r ${GOROOT}/src/cmd/internal/objabi ${PKG}

cp -r ${GOROOT}/src/internal/singleflight ${PKG}


find ${PWD}/pkg -type f -name '*.go' -exec sed -i 's/cmd\/go\/internal/github.com\/goproxyio\/goproxy\/pkg/g' {} +
find ${PWD}/pkg -type f -name '*.go' -exec sed -i 's/cmd\/internal/github.com\/goproxyio\/goproxy\/pkg/g' {} +
find ${PWD}/pkg -type f -name '*.go' -exec sed -i 's/internal/github.com\/goproxyio\/goproxy\/pkg/g' {} +