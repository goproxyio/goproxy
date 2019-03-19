#!/bin/sh

PKG="${PWD}/internal"
GOROOT="$(go env GOROOT)"
GOVERSION=`go version | awk -F ' ' '{print $3}'|grep '1.12'`
GTGO12=0
echo "ENV GOLANG VER: $GOVERSION"

if [[ "$GOVERSION" != "" ]]
then
    GTGO12=1
    echo "use 1.12 mode"
else
    echo "use 1.11 mode"
fi

mkdir -p "${PKG}"
cp -r "${GOROOT}/src/cmd/go/internal/"* "${PKG}"

cp -r "${GOROOT}/src/cmd/internal/browser" "${PKG}"
cp -r "${GOROOT}/src/cmd/internal/buildid" "${PKG}"
cp -r "${GOROOT}/src/cmd/internal/objabi" "${PKG}"
cp -r "${GOROOT}/src/cmd/internal/test2json" "${PKG}"

cp -r "${GOROOT}/src/internal/singleflight" "${PKG}"
cp -r "${GOROOT}/src/internal/testenv" "${PKG}"

if [[ "$GTGO12" = "1" ]]
then
    cp -r "${GOROOT}/src/internal/xcoff" "${PKG}"
    cp -r "${GOROOT}/src/internal/goroot" "${PKG}"
    cp -r "${GOROOT}/src/cmd/internal/sys" "${PKG}"
fi

find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/cmd\/go\/internal/github.com\/goproxyio\/goproxy\/internal/g' {} +
find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/cmd\/internal/github.com\/goproxyio\/goproxy\/internal/g' {} +
find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/internal\/singleflight/github.com\/goproxyio\/goproxy\/internal\/singleflight/g' {} +
find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/internal\/testenv/github.com\/goproxyio\/goproxy\/internal\/testenv/g' {} +

if [[ "$GTGO12" = "1" ]]
then
    find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/internal\/goroot/github.com\/goproxyio\/goproxy\/internal\/goroot/g' {} +
    find "${PKG}" -type f -name '*.go' -exec sed -i -e 's/internal\/xcoff/github.com\/goproxyio\/goproxy\/internal\/xcoff/g' {} +
fi
