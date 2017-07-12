# Timber Agent Makefile
#
# The contents of this file MUST be compatible with GNU Make 3.81,
# so do not use features or conventions introduced in later releases
# (for example, the ::= assignement operator)
#

distdir := $(CURDIR)/build
exec := timber-agent

.DEFAULT: all
.PHONY: clean test amd64-darwin amd64-linux

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
	go build -o $(bindir)/$(exec)

amd64-linux: arch := amd64
amd64-linux: os := linux
amd64-linux: target := $(arch)-$(os)
amd64-linux: destination := $(distdir)/$(target)/$(exec)
amd64-linux: bindir := $(destination)/bin
amd64-linux: export GOARCH=$(arch)
amd64-linux: export GOOS=$(os)
amd64-linux:
	mkdir -p $(bindir)
	go build -o $(bindir)/$(exec)

test:
	go test -v

clean:
	- rm -r $(distdir)
