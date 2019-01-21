#!/usr/bin/env bash

export GO111MODULE=on
export GOPROXY='http://127.0.0.1:8081'

datafile='test/testdata/get.txt'

while read -r line; do
	go get -v ${line}
done < "${datafile}"