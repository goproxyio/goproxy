FROM golang:alpine AS build

RUN apk add --no-cache -U make git mercurial subversion

COPY . /src/goproxy
RUN cd /src/goproxy &&\
    export CGO_ENABLED=0 &&\
    make

FROM golang:alpine

# Add tini
ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64 /usr/bin/tini
RUN chmod +x /usr/bin/tini

RUN apk add --no-cache -U git mercurial subversion

COPY --from=build /src/goproxy/bin/goproxy /goproxy

VOLUME /go

EXPOSE 8081

ENTRYPOINT ["/usr/bin/tini", "--", "/goproxy"]
CMD []
