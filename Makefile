.PHONY: build generate image clean

all: build

build: generate
	@go build -tags "netgo static_build" -installsuffix netgo -ldflags -w .

generate:
	@go generate

image:
	@docker build -t goproxy/goproxy .

clean:
	@git clean -f -d -X
