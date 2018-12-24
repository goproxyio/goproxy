FROM alpine:3.8
RUN apk add --no-cache git mercurial subversion bzr fossil
COPY bin/goproxy /usr/bin/goproxy