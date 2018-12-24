#!/bin/bash

export GO111MODULE=on
go generate
go mod tidy
# build
go build -v -mod readonly