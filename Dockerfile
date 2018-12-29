FROM golang:1.11 AS build

COPY ./ /goproxy

RUN cd /goproxy &&\
    export GO111MODULE=on &&\
    go generate &&\
    go mod tidy &&\
    go build

FROM alpine:3.8
RUN apk add --no-cache git mercurial subversion bzr fossil
COPY --from=build /goproxy/goproxy /bin/goproxy

EXPOSE 8081

CMD ["goproxy"]