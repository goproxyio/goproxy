FROM golang:alpine AS build

RUN apk add --no-cache -U make

COPY . /src/goproxy
WORKDIR /src/goproxy

ENV CGO_ENABLED=0

RUN make

FROM alpine:latest

RUN apk add --no-cache -U git mercurial subversion bzr fossil

COPY --from=build /src/goproxy/goproxy /goproxy

VOLUME /go

EXPOSE 8081

ENTRYPOINT ["/goproxy"]
CMD []
