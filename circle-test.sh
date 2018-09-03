#!/bin/bash

export GO111MODULE=on
go mod download
go mod verify
go test -timeout 60s -v ./...
# build
go build -v -mod readonly