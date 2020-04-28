# GOPROXY

[![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/goproxyio/goproxy)](https://goreportcard.com/report/github.com/goproxyio/goproxy)
[![GoDoc](https://godoc.org/github.com/goproxyio/goproxy?status.svg)](https://godoc.org/github.com/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Requirements
    It invokes the local go command to answer requests.
    The default cacheDir is GOPATH, you can set it up by yourself according to the situation.

## Build
    git clone https://github.com/goproxyio/goproxy.git
    cd goproxy
    make

## Started


### Proxy mode    
    
    ./bin/goproxy -listen=0.0.0.0:80 -cacheDir=/tmp/test

    If you run `go get -v pkg` in the proxy machine, should set a new GOPATH which is different from the old GOPATH, or mayebe deadlock.
    See the file test/get_test.sh.

### Router mode    

Use the -proxy flag switch to "Router mode", which 
implements route filter to routing private module 
or public module .

```
                                         direct
                      +----------------------------------> private repo
                      |
                 match|pattern
                      |
                  +---+---+           +----------+
go get  +-------> |goproxy| +-------> |goproxy.io| +---> golang.org/x/net
                  +-------+           +----------+
                 router mode           proxy mode
```

In Router mode, use the -exclude flag set pattern , direct to the repo which 
match the module path, pattern are matched to the full path specified, not only 
to the host component.

    ./bin/goproxy -listen=0.0.0.0:80 -cacheDir=/tmp/test -proxy https://goproxy.io -exclude "*.corp.example.com,rsc.io/private"

## Use docker image

    docker run -d -p80:8081 goproxy/goproxy

Use the -v flag to persisting the proxy module data (change ___cacheDir___ to your own dir):

    docker run -d -p80:8081 -v cacheDir:/go goproxy/goproxy

## Docker Compose

    docker-compose up

## Appendix

1. set `export GOPROXY=http://localhost` to enable your goproxy.
2. set `export GOPROXY=direct` to disable it.
