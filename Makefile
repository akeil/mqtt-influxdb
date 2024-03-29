BINARY  = ./cmd/mfx
BINDIR  = ./bin
ARMDIR  = $(BINDIR)/linux/arm
PKG     = ./pkg
QPKG	= github.com/akeil/mqtt-influxdb/pkg
VERSION = $(shell cat VERSION)
COMMIT	= $(shell git describe --always --long --dirty)

default: test build

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/mfx\
	 -ldflags="-X $(QPKG).Version=$(VERSION) -X $(QPKG).Commit=$(COMMIT)"\
	 $(BINARY)

arm:
	mkdir -p $(ARMDIR)
	env GOOS=linux GOARCH=arm go build -o $(ARMDIR)/mfx\
	 -ldflags="-X $(QPKG).Version=$(VERSION) -X $(QPKG).Commit=$(COMMIT)"\
	 $(BINARY)

install:
	go install $(BINARY)

fmt:
	gofmt -w pkg/*.go
	gofmt -w */*/*.go

test:
	go test $(PKG)

deps:
	go get -u github.com/eclipse/paho.mqtt.golang
	go get -u github.com/jmoiron/jsonq
