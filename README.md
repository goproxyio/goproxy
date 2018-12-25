
# GOPROXY [![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Build
    go generate
    go build

## Started
    
    ./goproxy -listen=0.0.0.0:80 -cacheDir=/cache

## Docker

    docker run --name goproxy -d -p80:8081 goproxyio/goproxy

Use the -v flag to persisting the proxy module data (change ___go_repo___ to your own dir):

    docker run --name goproxy -d -p80:8081 -v go_repo:/cache goproxyio/goproxy

## Docker Compose

    docker-compose up

## Appendix

1. set `$GOPROXY` to change your proxy or disable the proxy
