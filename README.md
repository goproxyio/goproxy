
# GOPROXY [![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Build

    go build

## Started
    
    ./goproxy -listen=0.0.0.0:80

## Docker

    docker run -it goproxyio/goproxy

Use the -v flag to persisting the proxy module data (change ___go_repo___ to your own dir):

    docker run -it -v go_repo:/go/pkg/mod/cache/download goproxyio/goproxy

## Docker Compose

    docker-compose up


