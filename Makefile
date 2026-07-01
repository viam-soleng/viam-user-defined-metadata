GO_BUILD_ENV := CGO_ENABLED=0
GO_BUILD_FLAGS := -ldflags="-s -w"
MODULE_BINARY := bin/user-defined-metadata

# Viam cloud build sets VIAM_TARGET_OS per target platform (see meta.json
# "build.arch"). Windows is cross-compiled and needs a .exe suffix plus a
# CGO-free build; the entrypoint in meta.json stays as the base name and Viam
# resolves the .exe on Windows. The linux/darwin arches are handled by `packages`.
ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -ldflags="-s -w" -tags no_cgo
	MODULE_BINARY = bin/user-defined-metadata.exe
endif

.PHONY: build lint update test module.tar.gz packages module all setup clean upload

build: $(MODULE_BINARY)

# Single-arch host/dev build (e.g. `make bin/user-defined-metadata`).
$(MODULE_BINARY): Makefile go.mod cmd/module/*.go models/*.go
	GOOS=$(VIAM_BUILD_OS) GOARCH=$(VIAM_BUILD_ARCH) $(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) ./cmd/module

lint:
	gofmt -s -w .

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	go test ./...

module.tar.gz: build
	tar -czf module.tar.gz $(MODULE_BINARY) meta.json

# `make packages` cross-compiles every architecture locally and produces
# one tarball per arch at bin/<goos>-<goarch>/module.tar.gz, ready for `make upload`.
# Each tarball contains `bin/user-defined-metadata` and `meta.json` at the top level,
# matching the entrypoint path declared in meta.json
packages: bin/linux-amd64/module.tar.gz bin/linux-arm64/module.tar.gz bin/darwin-arm64/module.tar.gz bin/windows-amd64/module.tar.gz

bin/%/module.tar.gz: Makefile go.mod cmd/module/*.go models/*.go meta.json
	@set -e; os=$$(echo $* | cut -d- -f1); arch=$$(echo $* | cut -d- -f2); \
	  workdir=bin/$*; bin=user-defined-metadata; ext=""; tags=""; \
	  if [ "$$os" = "windows" ]; then ext=".exe"; tags="-tags no_cgo"; fi; \
	  mkdir -p $$workdir/bin; \
	  echo ">> building $$os/$$arch -> $$workdir/bin/$$bin$$ext"; \
	  GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) $$tags -o $$workdir/bin/$$bin$$ext ./cmd/module; \
	  tar -czf $$workdir/module.tar.gz -C $$workdir bin/$$bin$$ext -C $(CURDIR) meta.json

module: test module.tar.gz

all: test packages

setup:
	go mod tidy

clean:
	rm -rf bin module.tar.gz

VERSION ?= 0.1.0

# Build every arch, then print the upload commands. Override the version with
# e.g. `make upload VERSION=1.2.3`.
upload: packages
	@echo viam module upload --version \"$(VERSION)\" --platform \"linux/amd64\" bin/linux-amd64/module.tar.gz
	@echo viam module upload --version \"$(VERSION)\" --platform \"linux/arm64\" bin/linux-arm64/module.tar.gz
	@echo viam module upload --version \"$(VERSION)\" --platform \"darwin/arm64\" bin/darwin-arm64/module.tar.gz
	@echo viam module upload --version \"$(VERSION)\" --platform \"windows/amd64\" bin/windows-amd64/module.tar.gz
