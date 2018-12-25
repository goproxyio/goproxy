#!/bin/bash

go env

export GO111MODULE=on
export GOROOT=/usr/local/go

go generate
go mod tidy
# build
go build -o bin/goproxy