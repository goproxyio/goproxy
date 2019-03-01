
# GOPROXY [![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Build
    go generate
    go build

## Started

    ./goproxy -listen=0.0.0.0:80 -cache_dir=/data
	 
You can also set proxy for goproxy by setting `HTTP_PROXY` and `HTTPS_PROXY`
	 
    HTTP_PROXY=$(proxy_server) HTTPS_PROXY=$(proxy_server) ./goproxy -listen=0.0.0.0:80 -cache_dir=/data

## Use docker image

    docker run -d -p80:8081 goproxy/goproxy

Use the -v flag to persisting the proxy module data (change ___cacheDir___ to your own dir):

    docker run -d -p80:8081 -v cache_dir:/go goproxy/goproxy

## Docker Compose

    docker-compose up

## Appendix

1. set `export GOPROXY=http://localhost` to enable your goproxy.
2. set `export GOPROXY=` to disable it.

