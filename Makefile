# Timber Agent Makefile
#
# The contents of this file MUST be compatible with GNU Make 3.81,
# so do not use features or conventions introduced in later releases
# (for example, the ::= assignement operator)
#
#
# P.S. David is well aware that this Makefile needs some DRY-ing up.
# Issue #23 covers it. If you'd like to help out, feel free!

distdir := $(CURDIR)/build
exec := timber-agent
version = $(shell cat VERSION)

.DEFAULT: all
.PHONY: clean test amd64-darwin amd64-linux amd64-darwin-tarball amd64-linux-tarball

all: amd64-darwin amd64-linux

amd64-darwin: arch := amd64
amd64-darwin: os := darwin
amd64-darwin: target := $(arch)-$(os)
amd64-darwin: destination := $(distdir)/$(target)/$(exec)
amd64-darwin: bindir := $(destination)/bin
amd64-darwin: export GOARCH=$(arch)
amd64-darwin: export GOOS=$(os)
amd64-darwin:
	mkdir -p $(bindir)
	go build -ldflags "-X main.version=$(version)" -o $(bindir)/$(exec)

amd64-linux: arch := amd64
amd64-linux: os := linux
amd64-linux: target := $(arch)-$(os)
amd64-linux: destination := $(distdir)/$(target)/$(exec)
amd64-linux: bindir := $(destination)/bin
amd64-linux: export GOARCH=$(arch)
amd64-linux: export GOOS=$(os)
amd64-linux:
	mkdir -p $(bindir)
	go build -ldflags "-X main.version=$(version)" -o $(bindir)/$(exec)

amd64-darwin-tarball: arch := amd64
amd64-darwin-tarball: os := darwin
amd64-darwin-tarball: target := $(arch)-$(os)
amd64-darwin-tarball: destination := $(distdir)/$(target)
amd64-darwin-tarball: amd64-darwin
	tar cjf $(distdir)/$(exec)-$(target)-$(version).tar.bz2 -C $(destination) $(exec)/

amd64-linux-tarball: arch := amd64
amd64-linux-tarball: os := linux
amd64-linux-tarball: target := $(arch)-$(os)
amd64-linux-tarball: destination := $(distdir)/$(target)
amd64-linux-tarball: amd64-linux
	tar cjf $(distdir)/$(exec)-$(target)-$(version).tar.bz2 -C $(destination) $(exec)/

test:
	@go test -v

clean:
	- rm -r $(distdir)
