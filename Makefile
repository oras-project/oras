PROJECT_PKG = github.com/deislabs/oras
CLI_EXE     = oras
CLI_PKG     = $(PROJECT_PKG)/cmd/oras
DEP         = $(GOPATH)/bin/dep
GIT_COMMIT  = $(shell git rev-parse HEAD)
GIT_TAG     = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

LDFLAGS = -w
ifdef VERSION
	LDFLAGS += -X $(PROJECT_PKG)/internal/version.BuildMetadata=$(VERSION)
endif
ifneq ($(GIT_TAG),)
	LDFLAGS += -X $(PROJECT_PKG)/internal/version.BuildMetadata=
endif
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitCommit=${GIT_COMMIT}
LDFLAGS += -X $(PROJECT_PKG)/internal/version.GitTreeState=${GIT_DIRTY}

.PHONY: test
test: vendor
	./scripts/test.sh

.PHONY: covhtml
covhtml:
	open .cover/coverage.html

.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: build
build: build-linux build-mac build-windows vendor

.PHONY: build-linux
build-linux: vendor
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac
build-mac: vendor
	GOARCH=amd64 CGO_ENABLED=0 GOOS=darwin go build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-windows
build-windows: vendor
	GOARCH=amd64 CGO_ENABLED=0 GOOS=windows go build -v --ldflags="$(LDFLAGS)" \
		-o bin/windows/amd64/$(CLI_EXE).exe $(CLI_PKG)

$(DEP):
	go get -u github.com/golang/dep/cmd/dep

# install vendored dependencies
vendor: Gopkg.lock
	$(DEP) ensure -v --vendor-only

# update vendored dependencies
Gopkg.lock: Gopkg.toml
	$(DEP) ensure -v --no-vendor

Gopkg.toml: $(DEP)
