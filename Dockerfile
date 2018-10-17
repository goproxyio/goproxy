FROM golang:1.11

COPY ./ /goproxy
WORKDIR /goproxy
RUN go build

CMD ["/goproxy/goproxy","-listen=0.0.0.0:8080"]

