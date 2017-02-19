BINARY = ./cmd/mfx
BINDIR = bin

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/mfx $(BINARY)

install:
	go install $(BINARY)
