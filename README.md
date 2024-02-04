# GOPROXY

[![CircleCI](https://circleci.com/gh/goproxyio/goproxy.svg?style=svg)](https://circleci.com/gh/goproxyio/goproxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/goproxyio/goproxy)](https://goreportcard.com/report/github.com/goproxyio/goproxy)
[![GoDoc](https://godoc.org/github.com/goproxyio/goproxy?status.svg)](https://godoc.org/github.com/goproxyio/goproxy)

A global proxy for go modules. see: [https://goproxy.io](https://goproxy.io)

## Requirements

This service invokes the local `go` command to answer requests.

The default `cacheDir` is `GOPATH`, you can set it up by yourself according to the situation.

## Build

```shell
git clone https://github.com/goproxyio/goproxy.git
cd goproxy
make
```

## Started

### Proxy mode    

```shell
./bin/goproxy -listen=0.0.0.0:80 -cacheDir=/tmp/test
```

If you run `go get -v pkg` in the proxy machine, you should set a new `GOPATH` which is different from the original `GOPATH`, or you may encounter a deadlock.

See [`test/get_test.sh`](./test/get_test.sh).

### Router mode    

```shell
./bin/goproxy -listen=0.0.0.0:80 -proxy https://goproxy.io
```

Use the `-proxy` flag combined with the `-exclude` flag to enable `Router mode`, which implements route filter to routing private modules or public modules.

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

In `Router mode`, use the `-exclude` flag to set a glob pattern. The glob will specify what packages should not try to resolve with the value of `-proxy`. Modules which match the `-exclude` pattern will resolve direct to the repo which 
matches the module path.

NOTE: Patterns are matched to the full path specified, not only to the host component.

```shell
./bin/goproxy -listen=0.0.0.0:80 -cacheDir=/tmp/test -proxy https://goproxy.io -exclude "*.corp.example.com,rsc.io/private"
```

### Private module authentication

Some private modules are gated behind `git` authentication. To resolve this, you can force git to rewrite the URL with a personal access token present for auth

```shell
git config --global url."https://${GITHUB_PERSONAL_ACCESS_TOKEN}@github.com/".insteadOf https://github.com/
```

This can be done for other git providers as well, following the same pattern

## Use docker image

```shell
docker run -d -p80:8081 goproxy/goproxy
```

Use the -v flag to persisting the proxy module data (change ___cacheDir___ to your own dir):

```
docker run -d -p80:8081 -v cacheDir:/go goproxy/goproxy
```

## Docker Compose

```shell
docker-compose up
```

## Kubernetes

Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: goproxy
  name: goproxy
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: goproxy
    spec:
      containers:
      - args:
        - -proxy
        - https://goproxy.io
        - -listen
        - 0.0.0.0:8081
        - -cacheDir
        - /tmp/test
        - -exclude
        - github.com/my-org/*
        image: goproxy/goproxy
        name: goproxy
        ports:
        - containerPort: 8081
        volumeMounts:
        - mountPath: /tmp/test
          name: goproxy
      volumes:
      - emptyDir:
          medium: Memory
          sizeLimit: 500Mi
        name: goproxy
```

Deployment (with gitconfig secret):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: goproxy
  name: goproxy
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: goproxy
    spec:
      containers:
      - args:
        - -proxy
        - https://goproxy.io
        - -listen
        - 0.0.0.0:8081
        - -cacheDir
        - /tmp/test
        - -exclude
        - github.com/my-org/*
        image: goproxy/goproxy
        name: goproxy
        ports:
        - containerPort: 8081
        volumeMounts:
        - mountPath: /tmp/test
          name: goproxy
        - mountPath: /root
          name: gitconfig
          readOnly: true
      volumes:
      - emptyDir:
          medium: Memory
          sizeLimit: 500Mi
        name: goproxy
      - name: gitconfig
        secret:
          secretName: gitconfig
---
apiVersion: v1
data:
  # NOTE: Encoded version of the following, replacing ${GITHUB_PERSONAL_ACCESS_TOKEN}
  # [url "https://${GITHUB_PERSONAL_ACCESS_TOKEN}@github.com/"]
  # insteadOf = https://github.com/
  .gitconfig: *****************************
kind: Secret
metadata:
  name: test
```

## Appendix

- If running locally, set `export GOPROXY=http://localhost[:PORT]` to use your goproxy.
- Set `export GOPROXY=direct` to directly access modules without your goproxy.
