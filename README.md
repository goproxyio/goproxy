
# GOPROXY [![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Build
``` sh
go generate
go build
```
## Started
    
    ./goproxy -listen=0.0.0.0:80 -root=/go

## Docker

    docker run --name goproxy -d -p80:8081 goproxyio/goproxy 

Use the -v flag to persisting the proxy module data (change ___go_repo___ to your own dir):

    docker run --name goproxy -d -p80:8081 -v go_repo:/go goproxyio/goproxy

## Docker Compose

    docker-compose up


