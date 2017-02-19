BINARY = ./cmd/mfx
BINDIR = bin
ARMDIR = bin/linux/arm

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/mfx $(BINARY)

arm:
	mkdir -p $(ARMDIR)
	env GOOS=linux GOARCH=arm go build -o $(ARMDIR)/mfx $(BINARY)

install:
	go install $(BINARY)
