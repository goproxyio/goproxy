FROM golang:1.11 AS build

env GO111MODULE on
env GOROOT /usr/local/go

COPY ./ /goproxy
WORKDIR /goproxy
RUN go generate
RUN go build

FROM buildpack-deps:stretch-scm
COPY --from=build /goproxy/goproxy /bin/goproxy
EXPOSE 8081

CMD ["goproxy"]