#!/bin/bash

export GO111MODULE=on

go generate
go mod tidy
go test -v `(go list ./... | grep "pkg/proxy")`
# build
go build -o bin/goproxy