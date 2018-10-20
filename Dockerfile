FROM golang:1.11 as builder

COPY . /goproxy
WORKDIR /goproxy

RUN CGO_ENABLED=0 go build -o goproxy .

FROM alpine:3.8

ENV GOPATH /go
RUN apk --no-cache add ca-certificates && \
    mkdir -p $GOPATH/pkg/mod/cache/download && \
    addgroup -g 99 appuser && \
    adduser -D -u 99 -G appuser appuser

USER appuser

WORKDIR /app
COPY --from=builder /goproxy/goproxy .

EXPOSE 8080
CMD ["/app/goproxy","-listen=0.0.0.0:8080"]
