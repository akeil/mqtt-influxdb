BINARY = ./cmd/mfx

build:
	go build $(BINARY)

install:
	go install $(BINARY)
