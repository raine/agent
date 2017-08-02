# Timber Agent Makefile
#
# The contents of this file MUST be compatible with GNU Make 3.81,
# so do not use features or conventions introduced in later releases
# (for example, the ::= assignement operator)
#
# The Makefile for github-release was used as a basis for this file.
# The specific version can be found here:
#   https://github.com/c4milo/github-release/blob/6d2edc2/Makefile
#
# P.S. David is well aware that this Makefile needs some DRY-ing up.
# Issue #23 covers it. If you'd like to help out, feel free!

build_dir := $(CURDIR)/build
dist_dir := $(CURDIR)/dist
s3_prefix := packages.timber.io/agent

exec := timber-agent
github_repo := timberio/agent
version = $(shell cat VERSION)

.DEFAULT_GOAL := dist

.PHONY: clean
clean: clean-build clean-dist

.PHONY: clean-build
clean-build:
	@echo "Removing build files"
	rm -rf $(build_dir)

.PHONY: clean-dist
clean-dist:
	@echo "Removing distribution files"
	rm -rf $(dist_dir)

.PHONY: build
build: clean-build
	@echo "Creating build directory"
	mkdir -p $(build_dir)
	@echo "Building targets"
	@gox -ldflags "-X main.version=$(version)" \
		-osarch="darwin/amd64" \
		-osarch="freebsd/amd64" \
		-osarch="linux/amd64" \
		-osarch="netbsd/amd64" \
		-osarch="openbsd/amd64" \
		-output "$(build_dir)/$(exec)-$(version)-{{.OS}}-{{.Arch}}/$(exec)/bin/$(exec)"

.PHONY: dist
dist: clean-dist build
	@echo "Creating distribution directory"
	mkdir -p $(dist_dir)
	@echo "Creating distribution archives"
	$(eval FILES := $(shell ls $(build_dir)))
	@for f in $(FILES); do \
		echo "Creating distribution archive for $$f"; \
		(cd $(build_dir)/$$f && tar -czf $(dist_dir)/$$f.tar.gz *); \
	done

.PHONY: release
release: dist
	@tag=v$(version); \
	changelog=$$(git show -s $$tag --pretty=tformat:%N | sed -e '1,3d'); \
	github-release $(github_repo) $$tag master $$changelog '$(dist_dir)/*';
	$(eval FILES := $(shell ls $(dist_dir)))
	@for exact_filename in $(FILES); do \
		rel=$$(echo $$exact_filename | sed "s/\.tar\.gz//"); \
		doublet=$$(echo $$rel | cut -d - -f 4,5); \
		latest_patch_version="$$(echo $(version) | cut -d . -f 1,2).x"; \
		latest_minor_version="$$(echo $(version) | cut -d . -f 1).x.x"; \
		latest_patch_filename=$$(echo $$exact_filename | sed "s/$(version)/$$latest_patch_version/"); \
		latest_minor_filename=$$(echo $$exact_filename | sed "s/$(version)/$$latest_minor_version/"); \
		exact_version_destination="s3://$(s3_prefix)/$(version)/$$doublet/$$exact_filename"; \
		latest_patch_destination="s3://$(s3_prefix)/$$latest_patch_version/$$doublet/$$latest_patch_filename"; \
		latest_minor_destination="s3://$(s3_prefix)/$$latest_minor_version/$$doublet/$$latest_minor_filename"; \
		echo "Uploading v$(version) as $(version) for $$arch ($$exact_filename) to S3 ($$exact_version_destination)"; \
		aws s3 cp $(dist_dir)/$$exact_filename $$exact_version_destination; \
		echo "Uploading v$(version) as $$latest_patch_version for $$arch ($$exact_filename) to S3 ($$latest_patch_destination)"; \
		aws s3 cp $(dist_dir)/$$exact_filename $$latest_patch_destination; \
		echo "Uploading v$(version) as $$latest_minor_version for $$arch ($$exact_filename) to S3 ($$latest_patch_destination)"; \
		aws s3 cp $(dist_dir)/$$exact_filename $$latest_minor_destination; \
	done

.PHONY: release
get-tools:
	go get github.com/golang/dep/cmd/dep
	go get github.com/c4milo/github-release
	go get github.com/mitchellh/gox
	go get github.com/jstemmer/go-junit-report

.PHONY: docker-image
docker-image: build
	docker build -t timberio/agent:$(version) --build-arg version=$(version) .

.PHONY: test
test:
	@go test -v
