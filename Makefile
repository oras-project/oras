PROJECT_PKG = oras.land/oras
CLI_EXE     = oras
CLI_PKG     = $(PROJECT_PKG)/cmd/oras
GIT_COMMIT  = $(shell git rev-parse HEAD)
GIT_TAG     = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY   = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

TARGET_OBJS ?= checksums.txt darwin_amd64.tar.gz darwin_arm64.tar.gz linux_amd64.tar.gz linux_arm64.tar.gz windows_amd64.tar.gz

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
test: vendor check-encoding
	echo "TODO: add unit tests"

.PHONY: covhtml
covhtml:
	open .cover/coverage.html

.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: build
build: build-linux build-mac build-windows

.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64 build-linux-arm-v7

.PHONY: build-linux-amd64
build-linux-amd64:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-arm64
build-linux-arm64:
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/arm64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-linux-arm-v7
build-linux-arm-v7:
	GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -v --ldflags="$(LDFLAGS)" \
		-o bin/linux/arm/v7/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac
build-mac: build-mac-arm64 build-mac-amd64

.PHONY: build-mac-amd64
build-mac-amd64:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=darwin go build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/amd64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-mac-arm64
build-mac-arm64:
	GOARCH=arm64 CGO_ENABLED=0 GOOS=darwin go build -v --ldflags="$(LDFLAGS)" \
		-o bin/darwin/arm64/$(CLI_EXE) $(CLI_PKG)

.PHONY: build-windows
build-windows:
	GOARCH=amd64 CGO_ENABLED=0 GOOS=windows go build -v --ldflags="$(LDFLAGS)" \
		-o bin/windows/amd64/$(CLI_EXE).exe $(CLI_PKG)

.PHONY: check-encoding
check-encoding:
	! find cmd internal -name "*.go" -type f -exec file "{}" ";" | grep CRLF

.PHONY: fix-encoding
fix-encoding:
	find cmd internal -type f -name "*.go" -exec sed -i -e "s/\r//g" {} +

.PHONY: vendor
vendor:
	GO111MODULE=on go mod vendor

.PHONY: fetch-dist
fetch-dist:
	mkdir -p _dist
	cd _dist && \
	for obj in ${TARGET_OBJS} ; do \
		curl -sSL -o oras_${VERSION}_$${obj} https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_$${obj} ; \
	done

.PHONY: sign
sign:
	for f in $$(ls _dist/*.{gz,txt} 2>/dev/null) ; do \
		gpg --armor --detach-sign $${f} ; \
	done
