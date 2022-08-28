mod-tidy:
	go mod tidy

build:
	go build -ldflags="-X main.version=$(shell git describe )"
