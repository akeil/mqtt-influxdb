BINARY  = ./cmd/mfx
BINDIR  = ./bin
ARMDIR  = $(BINDIR)/linux/arm
PKG     = ./mqttinflux
QPKG	= akeil.net/akeil/mqtt-influxdb/mqttinflux
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
	 -ldflags="-X $(QPKG).version=$(VERSION) -X $(QPKG).Commit=$(COMMIT)"\
	 $(BINARY)

install:
	go install $(BINARY)

test:
	go test $(PKG)
