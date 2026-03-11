VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
           -X github.com/ShuhaoZQGG/ccoverage/cmd.version=$(VERSION) \
           -X github.com/ShuhaoZQGG/ccoverage/cmd.commit=$(COMMIT) \
           -X github.com/ShuhaoZQGG/ccoverage/cmd.date=$(DATE)

.PHONY: build test clean menubar menubar-release app-bundle dmg demo

build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ccoverage .

test:
	CGO_ENABLED=0 go test ./...

clean:
	rm -f ccoverage
	rm -rf dist/ build/

menubar:
	cd menubar && swift build

menubar-release:
	cd menubar && swift build -c release --arch arm64 --arch x86_64

app-bundle: menubar-release
	./scripts/create-app-bundle.sh

dmg: app-bundle
	./scripts/create-dmg.sh

demo:
	vhs demo.tape
