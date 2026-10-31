SHELL := /bin/sh

VERSION := $(shell cat VERSION)
include packaging/revisions.env

PKGNAME := spark-store-tui
PKGROOT := packaging/debian/package-root
ARCH ?= $(shell dpkg --print-architecture 2>/dev/null || echo amd64)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/yxxbc/spark-store-tui/internal/version.Version=$(VERSION) \
	-X github.com/yxxbc/spark-store-tui/internal/version.Commit=$(COMMIT) \
	-X github.com/yxxbc/spark-store-tui/internal/version.Date=$(BUILD_DATE)

.PHONY: all check build source dist go-test go-build local-install clean release-check version

all: check build

check:
	sh -n $(PKGROOT)/usr/bin/sparkstore
	sh -n $(PKGROOT)/DEBIAN/postinst
	sh -n $(PKGROOT)/DEBIAN/postrm
	bash -n scripts/build-dist.sh
	bash -n scripts/install-sparkstore.sh
	bash -n scripts/publish-aur.sh
	bash -n scripts/sync-all.sh
	bash -n scripts/check-version.sh
	go test ./...
	go vet ./...
	go build ./...

# Build a native Debian package for the host architecture. The stage is copied
# into a Linux temporary directory first so this works when the checkout lives
# under WSL's /mnt/c (where Unix ownership and execute bits are unreliable).
build:
	scripts/build-dist.sh --skip-rpm --skip-source --deb-archs "$(ARCH)"

source:
	scripts/build-dist.sh --skip-deb --skip-rpm

# Build every release artifact: Deb, RPM, source archive, AUR files and
# SHA256SUMS. Requires a Linux host with dpkg-deb and rpmbuild.
dist:
	scripts/build-dist.sh

go-test:
	go test ./...
	go vet ./...

go-build:
	mkdir -p build
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o build/sparkstore ./cmd/sparkstore
	@probe=$$(mktemp -d build/.sparkstore-case.XXXXXX); \
	touch "$$probe/sparkstore"; \
	if [ ! -e "$$probe/SparkStore" ]; then \
		ln -sf sparkstore build/SparkStore; \
		ln -sf sparkstore build/SPARKSTORE; \
		ln -sf sparkstore build/spark-store-tui; \
	else \
		echo "case-insensitive build filesystem: use ./build/sparkstore"; \
	fi; \
	rm -rf "$$probe"

local-install: build
	sudo apt install ./dist/$(PKGNAME)_$(VERSION)-$(DEB_REVISION)_$(ARCH).deb

release-check:
	scripts/check-version.sh

version:
	@echo $(VERSION)

clean:
	rm -rf build dist
