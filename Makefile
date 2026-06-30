GO_BUILD_ENV   := CGO_ENABLED=0
GO_BUILD_FLAGS :=
MODULE_BINARY  := bin/user-defined-metadata

# Viam cloud build sets VIAM_TARGET_OS / VIAM_BUILD_OS / VIAM_BUILD_ARCH per
# target platform (see meta.json "build.arch"). Windows builds are cross-compiled
# and need a .exe suffix plus a CGO-free build. The entrypoint in meta.json stays
# as the base name (bin/user-defined-metadata) — Viam resolves the .exe on Windows.
ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -tags no_cgo
	MODULE_BINARY = bin/user-defined-metadata.exe
endif

$(MODULE_BINARY): Makefile go.mod cmd/module/*.go models/*.go
	GOOS=$(VIAM_BUILD_OS) GOARCH=$(VIAM_BUILD_ARCH) $(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) ./cmd/module

lint:
	gofmt -s -w .

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	go vet ./...

module.tar.gz: meta.json $(MODULE_BINARY)
	tar czf $@ meta.json $(MODULE_BINARY)

module: test module.tar.gz

all: test module.tar.gz

setup:
	go mod tidy

clean:
	rm -rf bin module.tar.gz
