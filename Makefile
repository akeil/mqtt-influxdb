BINARY = ./cmd/mfx
BINDIR = ./bin
ARMDIR = $(BINDIR)/linux/arm
PKG    = ./mqttinflux

default: test build

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/mfx $(BINARY)

arm:
	mkdir -p $(ARMDIR)
	env GOOS=linux GOARCH=arm go build -o $(ARMDIR)/mfx $(BINARY)

install:
	go install $(BINARY)

test:
	go test $(PKG)
