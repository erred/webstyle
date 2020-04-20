.PHONY: *
all: generate install

generate:
	go generate

install:
	go install go.seankhliao.com/webstyle/cmd/webrender
