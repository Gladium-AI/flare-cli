VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -s -w \
           -X github.com/paoloanzn/flare-cli/cmd.Version=$(VERSION) \
           -X github.com/paoloanzn/flare-cli/cmd.Commit=$(COMMIT) \
           -X github.com/paoloanzn/flare-cli/cmd.Date=$(DATE)

.PHONY: build install test lint clean

build:
	go build -ldflags '$(LDFLAGS)' -o flare .

install: build
	mkdir -p $(HOME)/.local/bin
	cp flare $(HOME)/.local/bin/flare
	@echo "Installed flare to $(HOME)/.local/bin/flare"
	@echo "Make sure $(HOME)/.local/bin is in your PATH"

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -f flare
